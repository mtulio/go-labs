package main

import (
	// "fmt"
	// "io"

	"fmt"
	"log"
	"os"
	"sync"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
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
		os.Exit(-1)
	}
}

func getUser(userName string) {
	svc := iam.New(session.New())
	input := &iam.GetUserInput{
		UserName: aws.String(userName),
	}

	result, err := svc.GetUser(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println(result)
}

func noop(i interface{}) {}

func main() {
	start := time.Now()
	svc := iam.New(session.New())
	input := &iam.ListUsersInput{}

	result, err := svc.ListUsers(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	//fmt.Println(result)
	// chUsers := make(chan *iam.User, 5)
	//chArnsExist := make(chan *string)
	chArnsFound := make(chan *string, 1)
	chArnsNotFound := make(chan *string, 1)
	//chDone := make(bool)
	// var done sync.WaitGroup
	arns := []string{}
	arnsNF := []string{}

	wp := &sync.WaitGroup{}
	wc := &sync.WaitGroup{}
	wn := &sync.WaitGroup{}

	wp.Add(len(result.Users))

	// wc.Add(consumerCount)
	fmt.Println(len(result.Users))
	// go func() {
	// 	select {
	// 	case user := <-chUsers:
	// 		go func(u *iam.User) {
	// 			defer done.Done()
	// 			//getUser(*u.UserName)
	// 			fmt.Printf("-> User= %s\n", *u.UserName)
	// 			go func() { chArnsFound <- u.Arn }()
	// 		}(user)
	// 	}
	// }()
	var totalArnsProcessed int = 0
	go func() {
		for arn := range chArnsFound {
			//fmt.Printf("-> UserARN= %s\n", *arn)
			arns = append(arns, *arn)
			totalArnsProcessed += 1
			//fmt.Println(totalArnsProcessed)
			wc.Done()
		}
	}()
	go func() {
		for arn := range chArnsNotFound {
			//fmt.Printf("-> UserARN= %s\n", *arn)
			arnsNF = append(arnsNF, *arn)
			//totalArnsProcessed += 1
			//fmt.Println(totalArnsProcessed)
			wn.Done()
		}
	}()
	for _, user := range result.Users {
		// done.Add(1)
		// fmt.Println(*u.UserName)
		// chUsers <- u
		go func(u *iam.User) {
			defer wp.Done()
			//getUser(*u.UserName)
			//fmt.Printf("-> User= %s\n", *u.UserName)
			//go func() { chArnsFound <- u.Arn }()
			listTagsInput := &iam.ListUserTagsInput{
				UserName: aws.String(*u.UserName),
			}
			listUsersTags, err := svc.ListUserTags(listTagsInput)
			if err != nil {
				fmt.Println(err.Error())
			} else {
				for _, tag := range listUsersTags.Tags {
					//fmt.Printf("- %s=%s \n", *tag.Key, *tag.Value)
					if (*tag.Key == *filterKey) && (*tag.Value == *filterValue) {
						wc.Add(1)
						chArnsFound <- u.Arn
					} else {
						wn.Add(1)
						chArnsNotFound <- u.Arn
					}
				}
			}

		}(user)
	}
	fmt.Println("Waiting...1")
	wp.Wait()
	fmt.Println("Waiting...2")
	fmt.Println(totalArnsProcessed)
	wc.Wait()
	fmt.Println(totalArnsProcessed)
	fmt.Println("Waiting...3")
	wn.Wait()
	fmt.Println(totalArnsProcessed)
	//done.Wait()
	// fmt.Println(len(result.Users))
	// fmt.Println(len(arns))
	elapsed := time.Since(start)
	log.Printf("took %s \n", elapsed)
	fmt.Printf("TotalUsers=[%d], Found=[%d] NoFound=[%d]\n", len(result.Users), len(arns), len(arnsNF))
	fmt.Println(arns)
}
