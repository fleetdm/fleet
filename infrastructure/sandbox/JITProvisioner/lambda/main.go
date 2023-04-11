package main

import (
	"github.com/akrylysov/algnhsa"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	flags "github.com/jessevdk/go-flags"

	//"github.com/juju/errors"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	_ "github.com/go-sql-driver/mysql"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
	"go.elastic.co/apm/module/apmgin/v2"
	_ "go.elastic.co/apm/v2"
)

type OptionsStruct struct {
	LambdaExecutionEnv string `long:"lambda-execution-environment" env:"AWS_EXECUTION_ENV"`
	LifecycleTable     string `long:"dynamodb-lifecycle-table" env:"DYNAMODB_LIFECYCLE_TABLE" required:"true"`
	LifecycleSFN       string `long:"lifecycle-sfn" env:"LIFECYCLE_SFN" required:"true"`
	FleetBaseURL       string `long:"fleet-base-url" env:"FLEET_BASE_URL" required:"true"`
	AuthorizationPSK   string `long:"authorization-psk" env:"AUTHORIZATION_PSK" required:"true"`
	MysqlSecret        string `long:"mysql-secret" env:"MYSQL_SECRET" required:"true"`
}

var options = OptionsStruct{}

func applyConfig(c *gin.Context, url, token string) (err error) {
	var client *service.Client
	if client, err = service.NewClient(url, false, "", ""); err != nil {
		log.Print(err)
		return
	}
	client.SetToken(token)

	buf, err := os.ReadFile("standard-query-library.yml")
	if err != nil {
		log.Print(err)
		return
	}
	specs, err := spec.GroupFromBytes(buf)
	if err != nil {
		return
	}
	logf := func(format string, a ...interface{}) {
		log.Printf(format, a...)
	}
	err = client.ApplyGroup(c, specs, "", logf, fleet.ApplySpecOptions{})
	if err != nil {
		return
	}
	return
}

type MysqlSecretEntry struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func clearActivitiesTable(c *gin.Context, id string) (err error) {
	// Get connection string
	svc := secretsmanager.New(session.New())
	sec, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(options.MysqlSecret),
	})
	if err != nil {
		log.Print(err)
		return
	}
	var secretEntry MysqlSecretEntry
	if err = json.Unmarshal([]byte(*sec.SecretString), &secretEntry); err != nil {
		log.Print(err)
		return
	}
	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", secretEntry.Username, secretEntry.Password, secretEntry.Endpoint, id)
	// Connect to db
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Print(err)
		return
	}
	defer db.Close()
	// truncate activities table
	_, err = db.ExecContext(c, "truncate activities;")
	if err != nil {
		log.Print(err)
		return
	}
	return
}

type LifecycleRecord struct {
	ID      string
	State   string
	RedisDB int `dynamodbav:"redis_db"`
	Token   string
}

func getExpiry(id string) (ret time.Time, err error) {
	var execArn arn.ARN
	var exec *sfn.DescribeExecutionOutput
	var input struct {
		WaitTime int `json:"waitTime"`
	}

	execArn, err = arn.Parse(options.LifecycleSFN)
	if err != nil {
		return
	}
	execArn.Resource = fmt.Sprintf("execution:%s:%s", strings.Split(execArn.Resource, ":")[1], id)

	exec, err = sfn.New(session.New()).DescribeExecution(&sfn.DescribeExecutionInput{
		ExecutionArn: aws.String(execArn.String()),
	})
	if err != nil {
		return
	}

	if err = json.Unmarshal([]byte(*exec.Input), &input); err != nil {
		return
	}
	var dur time.Duration
	if dur, err = time.ParseDuration(fmt.Sprintf("%ds", input.WaitTime)); err != nil {
		return
	}
	ret = exec.StartDate.Add(dur)
	return
}

func claimFleet(fleet LifecycleRecord, svc *dynamodb.DynamoDB) (err error) {
	log.Printf("Claiming instance: %+v", fleet)
	// Perform a conditional update to claim the item
	input := &dynamodb.UpdateItemInput{
		ConditionExpression: aws.String("#fleet_state = :v1"),
		TableName:           aws.String(options.LifecycleTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(fleet.ID),
			},
		},
		UpdateExpression:         aws.String("set #fleet_state = :v2"),
		ExpressionAttributeNames: map[string]*string{"#fleet_state": aws.String("State")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v1": {
				S: aws.String("unclaimed"),
			},
			":v2": {
				S: aws.String("claimed"),
			},
		},
	}
	if _, err = svc.UpdateItem(input); err != nil {
		return
	}
	return
}

func saveToken(fleet LifecycleRecord, svc *dynamodb.DynamoDB) (err error) {
	log.Printf("Saving Token: %+v", fleet)
	// Perform a conditional update to claim the item
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(options.LifecycleTable),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(fleet.ID),
			},
		},
		UpdateExpression:         aws.String("set #fleet_token = :v1"),
		ExpressionAttributeNames: map[string]*string{"#fleet_token": aws.String("Token")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v1": {
				S: aws.String(fleet.Token),
			},
		},
	}
	if _, err = svc.UpdateItem(input); err != nil {
		return
	}
	return
}

func getToken(id string, svc *dynamodb.DynamoDB) (token string, err error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(options.LifecycleTable),
		Key: map[string]*dynamodb.AttributeValue{"ID": &dynamodb.AttributeValue{
			S: aws.String(id),
		}},
	}

	var result *dynamodb.GetItemOutput
	if result, err = svc.GetItem(input); err != nil {
		return
	}
	var rec LifecycleRecord
	if err = dynamodbattribute.UnmarshalMap(result.Item, &rec); err != nil {
		return
	}
	token = rec.Token
	return
}

func getFleetInstance() (ret LifecycleRecord, err error) {
	log.Print("Getting fleet instance")
	svc := dynamodb.New(session.New())
	// Loop until we get one
	for {
		input := &dynamodb.QueryInput{
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":v1": {
					S: aws.String("unclaimed"),
				},
			},
			KeyConditionExpression:   aws.String("#fleet_state = :v1"),
			TableName:                aws.String(options.LifecycleTable),
			ExpressionAttributeNames: map[string]*string{"#fleet_state": aws.String("State")},
			IndexName:                aws.String("FleetState"),
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
			log.Print(err)
			continue
		}
		return
	}
}

func triggerSFN(id, expiry string) (err error) {
	var endTime time.Time
	log.Print("Triggering state machine")
	if endTime, err = time.Parse(time.RFC3339, expiry); err != nil {
		return
	}
	if int(endTime.Sub(time.Now()).Seconds()) < 0 {
		return errors.New("Expiry time is in the past")
	}
	sfnInStr, err := json.Marshal(struct {
		InstanceID string `json:"instanceID"`
		WaitTime   int    `json:"waitTime"`
	}{
		InstanceID: id,
		WaitTime:   int(endTime.Sub(time.Now()).Seconds()),
	})
	if err != nil {
		return
	}
	sfnIn := sfn.StartExecutionInput{
		Input:           aws.String(string(sfnInStr)),
		Name:            aws.String(id),
		StateMachineArn: aws.String(options.LifecycleSFN),
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
	Email             string `json:"email" validate:"required,email"`
	Name              string `json:"name" validate:"required"`
	SandboxExpiration string `json:"sandbox_expiration" validate:"required"`
	Password          string `json:"password" validate:"required"`
	Authorization     string `header:"Authorization" validate:"required"`
}
type NewFleetOutput struct {
	URL string
}

func NewFleet(c *gin.Context, in *NewFleetInput) (ret *NewFleetOutput, err error) {
	if in.Authorization != options.AuthorizationPSK {
		err = errors.New("Unauthorized")
		return
	}
	ret = &NewFleetOutput{}
	fleet, err := getFleetInstance()
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Creating fleet client")
	ret.URL = fmt.Sprintf("https://%s.%s", fleet.ID, options.FleetBaseURL)
	log.Print(ret.URL)
	client, err := service.NewClient(ret.URL, true, "", "")
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("Creating admin user")
	var token string
	if token, err = client.Setup(in.Email, in.Name, in.Password, "Fleet Sandbox"); err != nil {
		log.Print(err)
		return
	}
	fleet.Token = token
	log.Print("Triggering SFN to start teardown timer")
	if err = triggerSFN(fleet.ID, in.SandboxExpiration); err != nil {
		log.Print(err)
		return
	}
	log.Print("Applying basic config now that we have a user")
	if err = applyConfig(c, ret.URL, token); err != nil {
		log.Print(err)
		return
	}
	log.Print("Clearing activities table")
	if err = clearActivitiesTable(c, fleet.ID); err != nil {
		log.Print(err)
		return
	}
	log.Print("Saving admin token for addUser")
	if err = saveToken(fleet, dynamodb.New(session.New())); err != nil {
		log.Print(err)
		return
	}
	return
}

type AddUserInput struct {
	SandboxID     string `path:"SandboxID" validate:"required"`
	Authorization string `header:"Authorization" validate:"required"`
	Email         string `json:"email" validate:"required"`
	Password      string `json:"password" validate:"required"`
	Name          string `json:"name" validate:"required"`
}

type AddUserOutput struct{}

func AddUser(c *gin.Context, in *AddUserInput) (ret *AddUserOutput, err error) {
	if in.Authorization != options.AuthorizationPSK {
		err = errors.New("Unauthorized")
		return
	}
	client, err := service.NewClient(fmt.Sprintf("https://%s.%s", in.SandboxID, options.FleetBaseURL), true, "", "")
	if err != nil {
		log.Print(err)
		return
	}
	svc := dynamodb.New(session.New())
	token, err := getToken(in.SandboxID, svc)
	if err != nil {
		log.Print(err)
		return
	}
	client.SetToken(token)
	err = client.CreateUser(fleet.UserPayload{
		Password:                 ptr.String(in.Password),
		Email:                    ptr.String(in.Email),
		Name:                     ptr.String(in.Name),
		SSOEnabled:               &[]bool{false}[0],
		AdminForcedPasswordReset: &[]bool{false}[0],
		APIOnly:                  &[]bool{false}[0],
		GlobalRole:               ptr.String(fleet.RoleObserver),
		Teams:                    &[]fleet.UserTeam{},
	})
	return
}

type ExpiryInput struct {
	ID string `query:"id" validate:"required"`
}
type ExpiryOutput struct {
	Timestamp time.Time `json:"timestamp"`
}

func GetExpiry(c *gin.Context, in *ExpiryInput) (ret *ExpiryOutput, err error) {
	ret = &ExpiryOutput{}
	if ret.Timestamp, err = getExpiry(in.ID); err != nil {
		return
	}
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
	r.Use(cors.Default())
	f := fizz.NewFromEngine(r)
	infos := &openapi.Info{
		Title:       "Fleet Demo JITProvisioner",
		Description: "Provisions new Fleet instances upon request",
		Version:     "1.0.0",
	}
	f.GET("/openapi.json", nil, f.OpenAPI(infos, "json"))
	f.GET("/health", nil, tonic.Handler(Health, 200))
	f.POST("/new", nil, tonic.Handler(NewFleet, 200))
	f.GET("/expires", nil, tonic.Handler(GetExpiry, 200))
	f.POST("/addUser/:SandboxID", nil, tonic.Handler(AddUser, 200))
	algnhsa.ListenAndServe(r, nil)
}
