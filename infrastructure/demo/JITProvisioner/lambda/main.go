package main

import (
	"github.com/akrylysov/algnhsa"
	//"github.com/gin-contrib/cors" TODO: use cors
	"github.com/gin-gonic/gin"
	flags "github.com/jessevdk/go-flags"
	//"github.com/juju/errors"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
	"go.elastic.co/apm/module/apmgin/v2"
	_ "go.elastic.co/apm/v2"
	"log"
	"math/rand"
	"time"
)

type OptionsStruct struct {
	LambdaExecutionEnv string `long:"lambda-execution-environment" env:"AWS_EXECUTION_ENV"`
	LifecycleTable     string `long:"dynamodb-lifecycle-table" env:"DYNAMODB_LIFECYCLE_TABLE" required:"true"`
	LifecycleSFN       string `long:"lifecycle-sfn" env:"LIFECYCLE_SFN" required:"true"`
	FleetBaseURL       string `long:"fleet-base-url" env:"FLEET_BASE_URL" required:"true"`
}

var options = OptionsStruct{}

type LifecycleRecord struct {
	ID      string
	State   string
	RedisDB int `dynamodbav:"redis_db"`
}

func claimFleet(fleet LifecycleRecord, svc *dynamodb.DynamoDB) (err error) {
	// Perform a conditional update to claim the item
	input := &dynamodb.UpdateItemInput{
		ConditionExpression: aws.String("State = unclaimed"),
		TableName:           aws.String(options.LifecycleTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(fleet.ID),
			},
		},
		UpdateExpression: aws.String("set State = :s"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":S": {
				S: aws.String(fleet.ID),
			},
		},
	}
	if _, err = svc.UpdateItem(input); err != nil {
		return
	}
	return
}

func getFleetInstance() (ret LifecycleRecord, err error) {
	svc := dynamodb.New(session.New())
	// Loop until we get one
	for {
		input := &dynamodb.QueryInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":v1": {
					S: aws.String("unclaimed"),
				},
			},
			KeyConditionExpression: aws.String("State = :v1"),
			TableName:              aws.String(options.LifecycleTable),
		}

		var result *dynamodb.QueryOutput
		if result, err = svc.Query(input); err != nil {
			return
		}
		recs := []LifecycleRecord{}
		if err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &recs); err != nil {
			return
		}
		ret = recs[rand.Intn(len(recs))]
		if err = claimFleet(ret, svc); err != nil {
			continue
		}
		return
	}
}

func triggerSFN(id string) (err error) {
	sfnInStr, err := json.Marshal(struct {
		Id string
	}{
		Id: id,
	})
	if err != nil {
		return
	}
	sfnIn := sfn.StartExecutionInput{
		Input: aws.String(string(sfnInStr)),
		Name:  aws.String(options.LifecycleSFN),
	}
	_, err = sfn.New(session.New()).StartExecution(&sfnIn)
	return
}

type HealthInput struct{}
type HealthOutput struct {
	Message string `json:"message" description:"The status of the API." example:"The API is healthy"`
}

func Health(c *gin.Context, in *HealthInput) (ret *HealthOutput, err error) {
	ret = &HealthOutput{
		Message: "Healthy",
	}
	return
}

type NewFleetInput struct {
	Email    string `validate:"required"`
	Password string `validate:"required"`
}
type NewFleetOutput struct {
	URL string
}

func NewFleet(c *gin.Context, in *NewFleetInput) (ret *NewFleetOutput, err error) {
	fleet, err := getFleetInstance()
	if err != nil {
		return
	}
	ret.URL = fmt.Sprintf("%s.%s", fleet.ID, options.FleetBaseURL)
	client, err := service.NewClient(ret.URL, true, "", "", nil)
	if err != nil {
		return
	}
	client.Setup(in.Email, in.Email, in.Password, fleet.ID)
	triggerSFN(fleet.ID)
	return
}

func main() {
	rand.Seed(time.Now().Unix())
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

	r := gin.Default()
	r.Use(apmgin.Middleware(r))
	f := fizz.NewFromEngine(r)
	infos := &openapi.Info{
		Title:       "Fleet Demo JITProvisioner",
		Description: "Provisions new Fleet instances uppon request",
		Version:     "1.0.0",
	}
	f.GET("/openapi.json", nil, f.OpenAPI(infos, "json"))
	f.GET("/health", nil, tonic.Handler(Health, 200))
	f.POST("/new", nil, tonic.Handler(NewFleet, 200))
	algnhsa.ListenAndServe(r, nil)
}
