/*
  IAM List filter implementation
AWS' api has a limitation to filter a User by tags, so it needs to list
every user O(N) when it needs to find a user by tag. For that reason
accounts with large number of users can have troubles with throttles and slowly
running serial execution. A concurrency algorithm should help to fix this issue.
The proposal here is to validate the improvements in different types of AWS API calls.

Mode of operation tested:
- sync: serial execution / API calls to AWS
- async: concurrent execution / API calls when retrieving User's tags
- async-bucket: concurrent API calls with flow control of goroutines (using Leaky bucket)
to decrease the number of resources used by host.

Runtime/results - execution time:
- sync: 16m14.944660051s (~15MiB RAM)
- async: 11m34.321860026s (~11+GiB RAM ~90% CPU)
- async-bucket: 6m58.651769435s (~15MiB RAM)

*/

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

var (
	runMode     *string = flag.String("run-mode", "", "Run mode. Modes: sync | async | async-bucket")
	filterKey   *string = flag.String("filter-tag", "", "Tag Key to filter. required when filter")
	filterValue *string = flag.String("filter-value", "", "Tag Value to filter. required when filter")
)

func init() {
	flag.Parse()
	switch *runMode {
	case "sync":
		fmt.Println("Running as 'sync' mode")
	case "async":
		fmt.Println("Running as 'async' mode")
	case "async-bucket":
		fmt.Println("Running as 'async-bucket' mode (async list with Leaky Bucket implementation)")
	default:
		fmt.Printf("Mode [%s] not found. Value ones are: sync|async|async-bucket\n", *runMode)
		os.Exit(1)
	}
	if *filterKey == "" || *filterValue == "" {
		fmt.Println("missing --filter-(tag|value)")
		os.Exit(1)
	}
}

func main() {

	log.Println("Setting up AWS client")
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRetryer(func() aws.Retryer {
			retryer := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = 100
				o.MaxBackoff = 300 * time.Second
				o.RateLimiter = ratelimit.NewTokenRateLimit(10)
			})
			return retryer
		}))
	if err != nil {
		panic(err)
	}
	iamClient := iam.NewFromConfig(cfg)

	switch *runMode {
	case "sync":
		mainSync(iamClient)
		os.Exit(0)
	case "async":
		mainAsync(iamClient)
		os.Exit(0)
	case "async-bucket":
		mainAsyncBucket(iamClient)
		os.Exit(0)
	default:
		fmt.Printf("Mode %s not found\n", *runMode)
		os.Exit(1)
	}
}

/****************************************/
// Async with Leaky Bucket implementation
func mainAsyncBucket(iamClient *iam.Client) {
	wl := &sync.WaitGroup{} // main/list
	wf := &sync.WaitGroup{} // found
	wn := &sync.WaitGroup{} // not found
	we := &sync.WaitGroup{} // error
	wr := &sync.WaitGroup{} // retry
	chFound := make(chan *string, 30)
	chNFound := make(chan *string, 30)
	chErr := make(chan *types.User, 30)
	usersFoundArn := []string{}
	usersError := []*types.User{}
	totalNFound := 0
	totalListed := 0
	totalRetried := 0
	totalUsers := 0

	log.Println("Starting leaky bucket control")
	// limit concurrency to N
	semaphore := make(chan struct{}, 50)

	// have a max rate of N/sec
	rate := make(chan struct{}, 100)
	for i := 0; i < cap(rate); i++ {
		rate <- struct{}{}
	}

	// leaky bucket
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			_, ok := <-rate
			// if this isn't going to run indefinitely, signal
			// this to return by closing the rate channel.
			if !ok {
				return
			}
		}
	}()

	start := time.Now()

	log.Println("Starting data processors...")
	go func() {
		for arn := range chFound {
			usersFoundArn = append(usersFoundArn, *arn)
			wf.Done()
		}
	}()
	go func() {
		for range chNFound {
			//arnsNF = append(arnsNF, *arn)
			totalNFound += 1
			wn.Done()
		}
	}()
	go func() {
		for user := range chErr {
			usersError = append(usersError, user)
			we.Done()
		}
	}()
	defer func() {
		close(rate)
		close(chFound)
		close(chNFound)
		close(chErr)
	}()

	log.Println("Discovering users...")
	input := &iam.ListUsersInput{}
	paginator := iam.NewListUsersPaginator(iamClient, input, func(o *iam.ListUsersPaginatorOptions) {
		o.Limit = 10
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())

		if err != nil {
			log.Printf("error: %v", err)
			// return
			break
		}

		for _, user := range output.Users {
			totalUsers += 1
			wl.Add(1)
			go func(u types.User, cnt int) {
				defer wl.Done()

				// wait for the rate limiter
				rate <- struct{}{}

				// check the concurrency semaphore
				semaphore <- struct{}{}
				defer func() {
					<-semaphore
				}()

				// retrieve tags
				if (cnt % 500) == 0 {
					log.Printf("IAM Users finder: Still processing %d of %d\n", cnt, totalUsers)
				}
				listTagsInput := &iam.ListUserTagsInput{
					UserName: aws.String(*u.UserName),
				}
				// will wait until any channel is busy
				totalListed += 1
				listUsersTags, err := iamClient.ListUserTags(context.TODO(), listTagsInput)
				if err != nil {
					// ToDo: retrieve users that had errors to process tags
					fmt.Println(err.Error())
					we.Add(1)
					chErr <- &u
				} else {
					for _, tag := range listUsersTags.Tags {
						if (*tag.Key == *filterKey) && (*tag.Value == *filterValue) {
							wf.Add(1)
							chFound <- u.Arn
						} else {
							wn.Add(1)
							chNFound <- u.Arn
						}
					}
				}
			}(user, totalUsers)
		}
	}

	log.Println("Waiting User's tags lookup...")
	wl.Wait()

	// Need to process the error queue here
	// reprocessing error users
	log.Printf("Re-run failed users count: %d \n", len(usersError))
	for _, user := range usersError {
		wr.Add(1)
		listTagsInput := &iam.ListUserTagsInput{
			UserName: aws.String(*user.UserName),
		}
		// will wait until any channel is busy
		totalRetried += 1
		listUsersTags, err := iamClient.ListUserTags(context.TODO(), listTagsInput)
		if err != nil {
			fmt.Println(err.Error())
			//re-enqueue
			usersError = append(usersError, user)
		} else {
			for _, tag := range listUsersTags.Tags {
				if (*tag.Key == *filterKey) && (*tag.Value == *filterValue) {
					wf.Add(1)
					chFound <- user.Arn
				} else {
					wn.Add(1)
					chNFound <- user.Arn
				}
			}
		}
		wr.Done()
	}
	log.Println("Waiting retry processor...")
	wr.Wait()

	log.Println("Waiting User's found processor...")
	wf.Wait()

	log.Println("Waiting User's not found processor...")
	wn.Wait()

	log.Println("Waiting User's lookup error processor...")
	we.Wait()

	elapsed := time.Since(start)
	log.Printf("took %s \n", elapsed)
	log.Printf("TotalUsers=[%d], Found=[%d] NoFound=[%d] Errors=[%d]\n", totalUsers, len(usersFoundArn), totalNFound, len(usersError))
	log.Println(totalListed)
	log.Println(totalRetried)
}

/*********************/
// Sync implementation
func noop(i interface{}) {}

func mainSync(iamClient *iam.Client) {
	start := time.Now()
	fmt.Println("Starting Sync User index")
	input := &iam.ListUsersInput{}

	paginator := iam.NewListUsersPaginator(iamClient, input, func(o *iam.ListUsersPaginatorOptions) {
		o.Limit = 10
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())

		if err != nil {
			log.Printf("error: %v", err)
			return
		}

		for _, user := range output.Users {
			userName := *user.UserName
			//fmt.Printf("--- %s \n", userName)

			listTagsInput := &iam.ListUserTagsInput{
				UserName: aws.String(userName),
			}

			listUsersTags, err := iamClient.ListUserTags(context.TODO(), listTagsInput)
			if err != nil {
				fmt.Println(err.Error())
			} else {
				for _, tag := range listUsersTags.Tags {
					//fmt.Printf("- %s=%s \n", *tag.Key, *tag.Value)
					noop(tag)
				}
			}
		}
	}
	elapsed := time.Since(start)
	log.Printf("indexUsersSerial took %s \n", elapsed)
}

/**********************/
// Async implementation
func indexUserAsync(iamClient *iam.Client, userName string, errorCount int64) {
	//fmt.Printf("--- %s \n", userName)

	listTagsInput := &iam.ListUserTagsInput{
		UserName: aws.String(userName),
	}

	done := false
	for !done {
		listUsersTags, err := iamClient.ListUserTags(context.TODO(), listTagsInput)
		if err != nil {
			//fmt.Println("ERROR indexUserAsync(): " + err.Error())
			atomic.AddInt64(&errorCount, 1)
			// sleepTime := time.Duration(rand.Int()) * time.Second
			// time.Sleep(sleepTime)
		} else {
			for _, tag := range listUsersTags.Tags {
				//fmt.Printf("- %s=%s \n", *tag.Key, *tag.Value)
				noop(tag)
			}
			done = true
		}
	}

}

func mainAsync(iamClient *iam.Client) {
	start := time.Now()
	//fmt.Println("Starting Async User index")
	//fmt.Printf(" -- %T \n", iamClient)
	input := &iam.ListUsersInput{}

	paginator := iam.NewListUsersPaginator(iamClient, input, func(o *iam.ListUsersPaginatorOptions) {
		o.Limit = 1000
	})

	userNames := []string{}
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())

		if err != nil {
			log.Printf("error: %v", err)
			return
		}

		for _, user := range output.Users {
			userNames = append(userNames, *user.UserName)
		}
	}

	log.Printf("Indexing %d users", len(userNames))
	// time.Sleep(5 * time.Second)
	var errorCount int64 = 0
	wg := sync.WaitGroup{}
	for _, userName := range userNames {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			indexUserAsync(
				iamClient,
				u,
				errorCount,
			)
		}(userName)
	}
	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("indexUsersAsync took %s, with %d errors\n", elapsed, errorCount)
}
