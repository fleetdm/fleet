package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	flags "github.com/jessevdk/go-flags"
	"log"
)

type OptionsStruct struct {
	LambdaExecutionEnv string `long:"lambda-execution-environment" env:"AWS_EXECUTION_ENV"`
	LifecycleTable     string `long:"dynamodb-lifecycle-table" env:"DYNAMODB_LIFECYCLE_TABLE" required:"true"`
}

var options = OptionsStruct{}

type LifecycleRecord struct {
	State string
}

func getInstancesCount(c context.Context) (int64, int64, error) {
	log.Print("getInstancesCount")
	svc := dynamodb.New(session.New())
	// Example iterating over at most 3 pages of a Scan operation.
	var count, unclaimedCount int64
	err := svc.ScanPagesWithContext(
		c,
		&dynamodb.ScanInput{
			TableName: aws.String(options.LifecycleTable),
		},
		func(page *dynamodb.ScanOutput, lastPage bool) bool {
			count += *page.Count
			recs := []LifecycleRecord{}
			if err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &recs); err != nil {
				log.Print(err)
				return false
			}
			for _, i := range recs {
				if i.State == "unclaimed" {
					unclaimedCount++
				}
			}
			return true
		})
	if err != nil {
		return 0, 0, err
	}
	return count, unclaimedCount, nil
}

type NullEvent struct{}

func handler(ctx context.Context, name NullEvent) error {
	totalCount, unclaimedCount, err := getInstancesCount(ctx)
	if err != nil {
		log.Print(err)
		return err
	}
	svc := cloudwatch.New(session.New())
	log.Printf("Publishing %d, %d", totalCount, unclaimedCount)
	_, err = svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Fleet/sandbox"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("Type"),
						Value: aws.String("totalCount"),
					},
				},
				MetricName: aws.String("instances"),
				Value:      aws.Float64(float64(totalCount)),
				Unit:       aws.String(cloudwatch.StandardUnitCount),
			},
			&cloudwatch.MetricDatum{
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("Type"),
						Value: aws.String("unclaimedCount"),
					},
				},
				MetricName: aws.String("instances"),
				Value:      aws.Float64(float64(unclaimedCount)),
				Unit:       aws.String(cloudwatch.StandardUnitCount),
			},
		},
	})
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func main() {
	var err error
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Get config from environment
	parser := flags.NewParser(&options, flags.Default)
	if _, err = parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			return
		} else {
			log.Fatal(err)
		}
	}
	if options.LambdaExecutionEnv != "" {
		lambda.Start(handler)
	} else {
		if err = handler(context.Background(), NullEvent{}); err != nil {
			log.Fatal(err)
		}
	}
}
