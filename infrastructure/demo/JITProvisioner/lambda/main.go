package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	flags "github.com/jessevdk/go-flags"
	"github.com/juju/errors"
	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
	"log"
	"net/http"
	"github.com/fleetdm/fleet/infrastructure/demo/JITProvisioner/lambda/models"
	"github.com/fleetdm/fleet/infrastructure/demo/JITProvisioner/lambda/controllers"
)

type OptionsStruct struct {
	LambdaExecutionEnv string `long:"lambda-execution-environment" env:"AWS_EXECUTION_ENV"`
	LifecycleTable     string `long:"dynamodb-lifecycle-table" env:"DYNAMODB_LIFECYCLE_TABLE" required:"true"`
}

var options = OptionsStruct{}

type LifecycleRecord struct {
	ID    string
	State string
}

var ginLambda *httpadapter.HandlerAdapter

func NewRouter() (*fizz.Fizz, error) {
	engine := gin.New()
	engine.Use(cors.Default())
	fizzApp := fizz.NewFromEngine(engine)

	infos := &openapi.Info{
		Title:       "Fleet Demo JIT Provisioner",
		Description: "API for the Fleet Demo Environment",
		Version:     "1.0.0",
	}
	fizzApp.GET("/openapi.json", nil, fizzApp.OpenAPI(infos, "json"))
	group := fizzApp.Group("", "endpoints", "All of the endpoints.")
	group.GET("/healthcheck", []fizz.OperationOption{
		fizz.Summary("Checks API is healthy."),
		fizz.Response(fmt.Sprint(http.StatusInternalServerError), "Server Error", models.APIError{}, nil, nil),
	}, tonic.Handler(controllers.Healthcheck, http.StatusOK))
	group.POST("/new", []fizz.OperationOption{
		fizz.Summary("Get a new fleet demo instance"),
		fizz.Response(fmt.Sprint(http.StatusInternalServerError), "Server Error", models.APIError{}, nil, nil),
		fizz.Response(fmt.Sprint(http.StatusConflict), "No Capacity", models.APIError{}, nil, nil),
		fizz.Response(fmt.Sprint(http.StatusUnauthorized), "Incorrect Captcha", models.APIError{}, nil, nil),
		fizz.Response(fmt.Sprint(http.StatusBadRequest), "Invalid Input", models.APIError{}, nil, nil),
	}, tonic.Handler(controllers.NewDemo, http.StatusOK))

	if len(fizzApp.Errors()) != 0 {
		return nil, fmt.Errorf("fizz errors: %v", fizzApp.Errors())
	}
	tonic.SetErrorHook(errHook)
	return fizzApp, nil
}

func errHook(_ *gin.Context, e error) (int, interface{}) {
	code, msg := http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
	if _, ok := e.(tonic.BindError); ok {
		code, msg = http.StatusBadRequest, e.Error()
	} else {
		switch {
		case errors.IsBadRequest(e), errors.IsNotValid(e), errors.IsNotSupported(e), errors.IsNotProvisioned(e):
			code, msg = http.StatusBadRequest, e.Error()
		case errors.IsForbidden(e):
			code, msg = http.StatusForbidden, e.Error()
		case errors.IsMethodNotAllowed(e):
			code, msg = http.StatusMethodNotAllowed, e.Error()
		case errors.IsNotFound(e), errors.IsUserNotFound(e):
			code, msg = http.StatusNotFound, e.Error()
		case errors.IsUnauthorized(e):
			code, msg = http.StatusUnauthorized, e.Error()
		case errors.IsAlreadyExists(e):
			code, msg = http.StatusConflict, e.Error()
		case errors.IsNotImplemented(e):
			code, msg = http.StatusNotImplemented, e.Error()
		}
	}
	err := models.APIError{
		Message: msg,
	}
	return code, err
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return ginLambda.ProxyWithContext(ctx, req)
}

func getInstance(c *gin.Context) {
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

	r, err := NewRouter()
	if err != nil {
		log.Fatal(err)
	}

	if options.LambdaExecutionEnv == "AWS_Lambda_go1.x" {
		ginLambda = httpadapter.New(r)
		lambda.Start(handler)
	} else {
		srv := &http.Server{
			Addr:    ":8080",
			Handler: r,
		}
		srv.ListenAndServe()
	}
}
