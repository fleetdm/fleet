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
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	flags "github.com/jessevdk/go-flags"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type OptionsStruct struct {
	LambdaExecutionEnv           string `long:"lambda-execution-environment" env:"AWS_EXECUTION_ENV"`
	LifecycleTable               string `long:"dynamodb-lifecycle-table" env:"DYNAMODB_LIFECYCLE_TABLE" required:"true"`
	MaxInstances                 int64  `long:"max-instances" env:"MAX_INSTANCES" required:"true"`
	QueuedInstances              int64  `long:"queued-instances" env:"QUEUED_INSTANCES" required:"true"`
	FleetBaseURL                 string `long:"fleet-base-url" env:"FLEET_BASE_URL" required:"true"`
	InstallerBucket              string `long:"installer-bucket" env:"INSTALLER_BUCKET" required:"true"`
	MacOSDevIDCertificateContent string `long:"macos-dev-id-certificate-content" env:"MACOS_DEV_ID_CERTIFICATE_CONTENT" required:"true"`
	AppStoreConnectAPIKeyID      string `long:"app-store-connect-api-key-id" env:"APP_STORE_CONNECT_API_KEY_ID" required:"true"`
	AppStoreConnectAPIKeyIssuer  string `long:"app-store-connect-api-key-issuer" env:"APP_STORE_CONNECT_API_KEY_ISSUER" required:"true"`
	AppStoreConnectAPIKeyContent string `long:"app-store-connect-api-key-content" env:"APP_STORE_CONNECT_API_KEY_CONTENT" required:"true"`
}

var options = OptionsStruct{}

func FinishFleet(instanceID string) (err error) {
	log.Printf("Finishing instance: %s", instanceID)
	svc := dynamodb.New(session.New())
	// Perform a conditional update to claim the item
	input := &dynamodb.UpdateItemInput{
		ConditionExpression: aws.String("#fleet_state = :v1"),
		TableName:           aws.String(options.LifecycleTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(instanceID),
			},
		},
		UpdateExpression:         aws.String("set #fleet_state = :v2"),
		ExpressionAttributeNames: map[string]*string{"#fleet_state": aws.String("State")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v1": {
				S: aws.String("provisioned"),
			},
			":v2": {
				S: aws.String("unclaimed"),
			},
		},
	}
	if _, err = svc.UpdateItem(input); err != nil {
		return
	}
	return
}

func buildPackages(instanceID, enrollSecret string) (err error) {
	funcs := []func(packaging.Options) (string, error){
		packaging.BuildPkg,
		packaging.BuildDeb,
		packaging.BuildRPM,
		packaging.BuildMSI,
	}
	pkgopts := packaging.Options{
		FleetURL:                     fmt.Sprintf("https://%s.%s", instanceID, options.FleetBaseURL),
		EnrollSecret:                 enrollSecret,
		UpdateURL:                    "https://tuf.fleetctl.com",
		Identifier:                   "com.fleetdm.orbit",
		StartService:                 true,
		NativeTooling:                true,
		OrbitChannel:                 "stable",
		OsquerydChannel:              "stable",
		DesktopChannel:               "stable",
		OrbitUpdateInterval:          15 * time.Minute,
		Notarize:                     true,
		MacOSDevIDCertificateContent: options.MacOSDevIDCertificateContent,
		AppStoreConnectAPIKeyID:      options.AppStoreConnectAPIKeyID,
		AppStoreConnectAPIKeyIssuer:  options.AppStoreConnectAPIKeyIssuer,
		AppStoreConnectAPIKeyContent: options.AppStoreConnectAPIKeyContent,
	}
	store, err := s3.NewInstallerStore(config.S3Config{
		Bucket: options.InstallerBucket,
		Prefix: instanceID,
	})

	// Build non-desktop
	for _, buildFunc := range funcs {
		var filename string
		filename, err = buildFunc(pkgopts)
		if err != nil {
			log.Print(err)
			return
		}
		var r *os.File
		r, err = os.Open(filename)
		defer r.Close()
		if err != nil {
			return err
		}
		_, err = store.Put(context.Background(), fleet.Installer{
			EnrollSecret: enrollSecret,
			Kind:         filepath.Ext(filename)[1:],
			Desktop:      pkgopts.Desktop,
			Content:      r,
		})
		if err != nil {
			return
		}
	}

	// Build desktop
	pkgopts.Desktop = true
	for _, buildFunc := range funcs {
		var filename string
		filename, err = buildFunc(pkgopts)
		if err != nil {
			log.Print(err)
			return
		}
		var r *os.File
		r, err = os.Open(filename)
		defer r.Close()
		if err != nil {
			return err
		}
		_, err = store.Put(context.Background(), fleet.Installer{
			EnrollSecret: enrollSecret,
			Kind:         filepath.Ext(filename)[1:],
			Desktop:      pkgopts.Desktop,
			Content:      r,
		})
		if err != nil {
			return
		}
	}
	return FinishFleet(instanceID)
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

func runTerraform(workspace string, redis_database int, enrollSecret string) error {
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
		"-var",
		fmt.Sprintf("enroll_secret=%s", enrollSecret),
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
	has_init := false
	// deploy terraform to initialize everything
	// If there's an error during spinup, the program exits, so it either makes progress or fails completely, never running forever
	for min(options.MaxInstances-totalCount, options.QueuedInstances-unclaimedCount) > 0 {
		if !has_init {
			has_init = true
			if err := initTerraform(); err != nil {
				return err
			}
		}
		redisDatabase, err := getRedisDatabase()
		if err != nil {
			return err
		}
		enrollSecret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
		if err != nil {
			return err
		}
		instanceID := fmt.Sprintf("t%s", uuid.New().String()[:8])
		// This should fail if the instance id we pick already exists since it will collide with the primary key in dynamodb
		// This also actually puts the claim in place
		if err := runTerraform(instanceID, redisDatabase, enrollSecret); err != nil {
			return err
		}
		if err = buildPackages(instanceID, enrollSecret); err != nil {
			return err
		}

		// Refresh the count variables
		totalCount, unclaimedCount, err = getInstancesCount()
		if err != nil {
			return err
		}
		if totalCount >= options.MaxInstances {
			return nil
		}
		if unclaimedCount >= options.QueuedInstances {
			return nil
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
