// https://go.dev/play/p/ZrTPLcdeDF
// https://en.wikipedia.org/wiki/Leaky_bucket

package main

import (
	// "fmt"
	// "io"

	"context"
	"fmt"
	"log"
	"os"
	"sync"
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
	op          *string = flag.String("operation", "list", "Operations: list | filter")
	filterKey   *string = flag.String("filter-tag", "", "Tag Key to filter. required when filter")
	filterValue *string = flag.String("filter-value", "", "Tag Value to filter. required when filter")
)

func init() {
	flag.Parse()
	switch *op {
	case "list":
		fmt.Println("Running as 'sync' mode")
	case "filter":
		if *filterKey == "" || *filterValue == "" {
			fmt.Println("missing --filter-(tag|value)")
			os.Exit(1)
		}
	default:
		fmt.Printf("Operation %s not found\n", *op)
		os.Exit(1)
	}
}

func main() {
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

	log.Println("Setting up AWS client")
	start := time.Now()

	// IAM client setup
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

	input := &iam.ListUsersInput{}
	paginator := iam.NewListUsersPaginator(iamClient, input, func(o *iam.ListUsersPaginatorOptions) {
		o.Limit = 10
	})
	totalUsers := 0
	log.Println("Discovering users...")
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
	for _, user := range usersError {
		log.Printf("Processing error queue len: %d \n", len(usersError))
		wr.Add(1)
		listTagsInput := &iam.ListUserTagsInput{
			UserName: aws.String(*user.UserName),
		}
		// will wait until any channel is busy
		totalRetried += 1
		listUsersTags, err := iamClient.ListUserTags(context.TODO(), listTagsInput)
		if err != nil {
			// ToDo: retrieve users that had errors to process tags
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
	// log.Println(totalArnsProcessed)
	wf.Wait()
	// log.Println(totalArnsProcessed)
	log.Println("Waiting User's not found processor...")
	wn.Wait()
	// log.Println(totalArnsProcessed)
	log.Println("Waiting User's lookup error processor...")
	we.Wait()

	elapsed := time.Since(start)
	log.Printf("took %s \n", elapsed)
	log.Printf("TotalUsers=[%d], Found=[%d] NoFound=[%d] Errors=[%d]\n", totalUsers, len(usersFoundArn), totalNFound, len(usersError))
	log.Println(totalListed)
	log.Println(totalRetried)
}
