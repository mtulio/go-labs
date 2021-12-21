package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	flag "github.com/spf13/pflag"
)

var (
	runMode *string = flag.String("run-mode", "async", "Run mode. Modes: sync | async")
)

func main() {
	fmt.Printf("Setup client retry strategy...\n")

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

	flag.Parse()
	iamClient := iam.NewFromConfig(cfg)
	switch *runMode {
	case "sync":
		fmt.Println("Running as 'sync' mode")
		indexUsersSync(iamClient)
	case "async":
		fmt.Println("Running as 'sync' mode")
		indexUsersAsync(iamClient)
	default:
		fmt.Printf("Mode %s not found\n", *runMode)
	}

}

var onErrorSleep = 5

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

func noop(i interface{}) {}

func indexUsersAsync(iamClient *iam.Client) {
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
	var errorCount int64
	errorCount = 0
	wg := sync.WaitGroup{}
	for _, userName := range userNames {
		wg.Add(1)
		go func() {
			defer wg.Done()
			indexUserAsync(
				iamClient,
				userName,
				errorCount,
			)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("indexUsersAsync took %s and %d \n", elapsed, errorCount)
}

func indexUsersSync(iamClient *iam.Client) {
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
