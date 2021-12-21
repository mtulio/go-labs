package main

import (
	// "fmt"
	// "io"

	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

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

func main() {
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
	chArnsFound := make(chan *string)
	//chDone := make(bool)
	// var done sync.WaitGroup
	arns := []string{}

	wp := &sync.WaitGroup{}
	wc := &sync.WaitGroup{}

	wp.Add(len(result.Users))
	wc.Add(len(result.Users))
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
			fmt.Printf("-> UserARN= %s\n", *arn)
			arns = append(arns, *arn)
			totalArnsProcessed += 1
			fmt.Println(totalArnsProcessed)
			wc.Done()
		}
	}()
	for _, user := range result.Users {
		// done.Add(1)
		// fmt.Println(*u.UserName)
		// chUsers <- u
		go func(u *iam.User) {
			defer wp.Done()
			//getUser(*u.UserName)
			fmt.Printf("-> User= %s\n", *u.UserName)
			go func() { chArnsFound <- u.Arn }()
		}(user)
	}
	fmt.Println("Waiting...1")
	wp.Wait()
	fmt.Println("Waiting...2")
	fmt.Println(totalArnsProcessed)
	wc.Wait()
	fmt.Println(totalArnsProcessed)
	//done.Wait()
	// fmt.Println(len(result.Users))
	// fmt.Println(len(arns))
}
