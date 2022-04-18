package main

import (
	"context"
	"log"

	"database/sql"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	flags "github.com/jessevdk/go-flags"

	_ "github.com/go-sql-driver/mysql"
)

var ginLambda *ginadapter.GinLambda

type OptionsStruct struct {
	LambdaExecutionEnv string `long:"lambda-execution-environment" env:"AWS_EXECUTION_ENV"`
	MysqlUrlSecret     string `long:"mysql-url-secret" env:"MYSQL_URL_SECRET" required:"true"`
}

var options = OptionsStruct{}
var DB *sql.DB

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return ginLambda.ProxyWithContext(ctx, req)
}

func getSecretValue(secretId string) (string, error) {
	svc := secretsmanager.New(session.New())
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretId),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		return "", err
	}

	return *result.SecretString, nil
}

func main() {
	var err error
	var mysqlConnString string
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
	if mysqlConnString, err = getSecretValue(options.MysqlUrlSecret); err != nil {
		log.Fatal(err)
	}
	if DB, err = sql.Open("mysql", mysqlConnString); err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	if options.LambdaExecutionEnv == "AWS_Lambda_go1.x" {
		lambda.Start(Handler)
		ginLambda = ginadapter.New(r)
	} else {
		r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	}
}
