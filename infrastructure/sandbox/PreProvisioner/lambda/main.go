package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/google/uuid"
	flags "github.com/jessevdk/go-flags"
	"log"
	"math/rand"
	"os"
	"os/exec"
)

type OptionsStruct struct {
	LambdaExecutionEnv string `long:"lambda-execution-environment" env:"AWS_EXECUTION_ENV"`
	LifecycleTable     string `long:"dynamodb-lifecycle-table" env:"DYNAMODB_LIFECYCLE_TABLE" required:"true"`
	MaxInstances       int64  `long:"max-instances" env:"MAX_INSTANCES" required:"true"`
	QueuedInstances    int64  `long:"queued-instances" env:"QUEUED_INSTANCES" required:"true"`
}

var options = OptionsStruct{}

func buildPackages() {

}

type LifecycleRecord struct {
	ID    string
	State string
}

func getInstancesCount() (int64, int64, error) {
	log.Print("getInstancesCount")
	svc := dynamodb.New(session.New())
	// Example iterating over at most 3 pages of a Scan operation.
	var count, unclaimedCount int64
	err := svc.ScanPages(
		&dynamodb.ScanInput{
			TableName: aws.String(options.LifecycleTable),
		},
		func(page *dynamodb.ScanOutput, lastPage bool) bool {
			log.Print(page)
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

func min(a, b int64) int64 {
	// I really have to implement this myself?
	if a < b {
		return a
	}
	return b
}

func runCmd(args []string) error {
	cmd := exec.Cmd{
		Path:   "/build/terraform",
		Dir:    "/build/deploy_terraform",
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Args:   append([]string{"/build/terraform"}, args...),
	}
	log.Printf("%+v\n", cmd)
	return cmd.Run()
}

func initTerraform() error {
	err := runCmd([]string{
		"init",
		"-backend-config=backend.conf",
	})
	return err
}

func runTerraform(workspace string, redis_database int) error {
	err := runCmd([]string{
		"workspace",
		"new",
		workspace,
	})
	if err != nil {
		return err
	}
	err = runCmd([]string{
		"apply",
		"-auto-approve",
		"-no-color",
		"-var",
		fmt.Sprintf("redis_database=%d", redis_database),
	})
	return err
}

func idExists(id int) (bool, error) {
	svc := dynamodb.New(session.New())
	input := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v1": {
				N: aws.String(fmt.Sprintf("%d", id)),
			},
		},
		KeyConditionExpression: aws.String("redis_db = :v1"),
		TableName:              aws.String(options.LifecycleTable),
		IndexName:              aws.String("RedisDatabases"),
	}

	result, err := svc.Query(input)
	if err != nil {
		return false, err
	}
	return *result.Count != 0, nil
}

func getRedisDatabase() (int, error) {
	for {
		ret := rand.Intn(65536)
		exists, err := idExists(ret)
		if err != nil {
			return 0, err
		}
		if !exists {
			return ret, nil
		}
	}
}

func handler(ctx context.Context, name NullEvent) error {
	// check if we need to do anything
	totalCount, unclaimedCount, err := getInstancesCount()
	if err != nil {
		return err
	}
	if totalCount >= options.MaxInstances {
		return nil
	}
	if unclaimedCount >= options.QueuedInstances {
		return nil
	}
	numToReady := min(options.MaxInstances-totalCount, options.QueuedInstances-unclaimedCount)
	// deploy terraform to initialize everything
	for i := int64(0); i < numToReady; i++ {
		if i == 0 {
			if err := initTerraform(); err != nil {
				return err
			}
		}
		redisDatabase, err := getRedisDatabase()
		if err != nil {
			return err
		}
		if err := runTerraform(fmt.Sprintf("t%s", uuid.New().String()[:8]), redisDatabase); err != nil {
			return err
		}
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
	if options.LambdaExecutionEnv == "AWS_Lambda_go1.x" {
		lambda.Start(handler)
	} else {
		if err = handler(context.Background(), NullEvent{}); err != nil {
			log.Fatal(err)
		}
	}
}
