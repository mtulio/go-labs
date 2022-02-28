package watcher

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
)

type TargetGroupWatcher struct {
	options *TGWatcherOptions

	cliSvc   *elbv2.ELBV2
	cliInput *elbv2.DescribeTargetHealthInput
}

type TGWatcherOptions struct {
	ARN      string
	Interval time.Duration
	Metric   *metric.MetricsHandler
	Event    *event.EventHandler
}

func NewTargetGroupWatcher(op *TGWatcherOptions) (*TargetGroupWatcher, error) {

	tgw := TargetGroupWatcher{
		options: op,
	}
	//sess := session.Must(session.NewSession())
	svc := elbv2.New(session.New())
	input := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(op.ARN),
	}
	tgw.cliSvc = svc
	tgw.cliInput = input
	return &tgw, nil
}

func (tg *TargetGroupWatcher) Start() {
	// log.Println("START")
	for {
		result, err := tg.cliSvc.DescribeTargetHealth(tg.cliInput)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case elbv2.ErrCodeInvalidTargetException:
					fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
				case elbv2.ErrCodeTargetGroupNotFoundException:
					fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
				case elbv2.ErrCodeHealthUnavailableException:
					fmt.Println(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			log.Println("Returning eith error...")
			return
		}
		// fmt.Println(result)
		healthCount := 0
		unhealthyCount := 0
		for _, d := range result.TargetHealthDescriptions {
			// fmt.Println(*d.Target.Id)
			if *d.TargetHealth.State == "healthy" {
				healthCount += 1
				continue
			}
			unhealthyCount += 1
			// log.Println(*d.TargetHealth)
		}
		// log.Println(healthCount)
		// log.Println(len(result.TargetHealthDescriptions))
		tg.options.Metric.TargetHealthy = (unhealthyCount == 0)
		tg.options.Metric.TargetHealthCount = uint64(healthCount)
		tg.options.Metric.TargetUnhealthCount = uint64(unhealthyCount)
		time.Sleep(1 * time.Second)
	}
}

type TGTargetHealthy map[string]int

func (tg *TargetGroupWatcher) Collect() *TGTargetHealthy {
	m := make(TGTargetHealthy)

	result, err := tg.cliSvc.DescribeTargetHealth(tg.cliInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeInvalidTargetException:
				fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeHealthUnavailableException:
				fmt.Println(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return &m
	}

	for _, d := range result.TargetHealthDescriptions {
		healthy := 0
		if *d.TargetHealth.State == "healthy" {
			healthy = 1
		}
		m[*(d.Target.Id)] = healthy
	}

	return &m
}
