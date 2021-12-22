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
	// var wg sync.WaitGroup
	wg := &sync.WaitGroup{}
	wc := &sync.WaitGroup{}
	wn := &sync.WaitGroup{}
	we := &sync.WaitGroup{}
	chArnsFound := make(chan *string, 30)
	chArnsNF := make(chan *string, 30)
	chArnsErr := make(chan *string, 30)
	arnsFound := []string{}
	//arnsNF := []string{}
	arnsErr := []string{}
	totalArnsNF := 0
	totalListing := 0

	log.Println("Starting leaky bucket control")
	// log.Println("01")
	// limit concurrency to 5
	semaphore := make(chan struct{}, 50)

	// have a max rate of 10/sec
	rate := make(chan struct{}, 100)
	for i := 0; i < cap(rate); i++ {
		rate <- struct{}{}
	}
	// log.Println("02")
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
	// log.Println("03")
	log.Println("Setting up AWS client")
	start := time.Now()

	// IAM
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

	log.Println("Starting ARNs processors...")
	go func() {
		for arn := range chArnsFound {
			arnsFound = append(arnsFound, *arn)
			wc.Done()
		}
	}()
	go func() {
		for range chArnsNF {
			//arnsNF = append(arnsNF, *arn)
			totalArnsNF += 1
			wn.Done()
		}
	}()
	go func() {
		for arn := range chArnsErr {
			arnsErr = append(arnsErr, *arn)
			we.Done()
		}
	}()
	defer func() {
		close(rate)
		close(chArnsFound)
		close(chArnsNF)
		close(chArnsErr)
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
			// if (totalUsers % 100) == 0 {
			// 	log.Println(totalUsers)
			// }
			//fmt.Printf("[%d] Starting.... %s\n", totalUsers, *user.UserName)
			// fmt.Printf("Starting.... %s\n", *u.UserName)

			wg.Add(1)
			go func(u types.User, cnt int) {
				//fmt.Printf("[%d]> starting concurrency.... %s\n", totalUsers, *u.UserName)
				defer wg.Done()
				// wait for the rate limiter
				rate <- struct{}{}

				// check the concurrency semaphore
				semaphore <- struct{}{}
				defer func() {
					<-semaphore
				}()
				//fmt.Printf("[%d]> bucket released... %s\n", cnt, *u.UserName)
				// wait for the rate limiter
				// fmt.Printf("Listing tags for user %s\n", *u.UserName)
				// retrieve tags
				if (cnt % 500) == 0 {
					log.Printf("IAM Users finder: Still processing %d of %d\n", cnt, totalUsers)
				}
				listTagsInput := &iam.ListUserTagsInput{
					UserName: aws.String(*u.UserName),
				}
				// will wait until any channel is busy
				totalListing += 1
				listUsersTags, err := iamClient.ListUserTags(context.TODO(), listTagsInput)
				if err != nil {
					fmt.Println(err.Error())
					we.Add(1)
					chArnsErr <- u.Arn
				} else {
					for _, tag := range listUsersTags.Tags {
						//fmt.Printf("- %s=%s \n", *tag.Key, *tag.Value)
						if (*tag.Key == *filterKey) && (*tag.Value == *filterValue) {
							wc.Add(1)
							chArnsFound <- u.Arn
						} else {
							wn.Add(1)
							chArnsNF <- u.Arn
						}
					}
				}
			}(user, totalUsers)
		}
	}

	log.Println("Waiting User's tag lookup...")
	wg.Wait()
	// close(rate)

	// dur := time.Since(start)
	// log.Println("duration=", dur)

	log.Println("Waiting User's found processor...")
	// log.Println(totalArnsProcessed)
	wc.Wait()
	// log.Println(totalArnsProcessed)
	log.Println("Waiting User's not found processor...")
	wn.Wait()
	// log.Println(totalArnsProcessed)
	log.Println("Waiting User's lookup error processor...")
	we.Wait()

	elapsed := time.Since(start)
	log.Printf("took %s \n", elapsed)
	log.Printf("TotalUsers=[%d], Found=[%d] NoFound=[%d] Errors=[%d]\n", totalUsers, len(arnsFound), totalArnsNF, len(arnsErr))
	log.Println(arnsFound)
	log.Println(totalListing)
}
