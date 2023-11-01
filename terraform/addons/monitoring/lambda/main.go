/*
This script is intended to be used with AWS Lambda to monitor the various
crons that live inside of Fleet.

We will check to see if there are recent updates from the crons in the
following table:

    - cron_stats

If we have an old/incomplete run in cron_stats or if we are missing a
cron entry entirely, throw an alert to an SNS topic.

Currently tested crons:

    - cleanups_then_aggregation
    - vulnerabilities

*/

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/go-sql-driver/mysql"
	flags "github.com/jessevdk/go-flags"
)

type NullEvent struct{}

type OptionsStruct struct {
	LambdaRuntimeAPI   string `long:"lambda-runtime-api" env:"AWS_LAMBDA_RUNTIME_API"`
	SNSTopicArns       string `long:"sns-topic-arn" env:"SNS_TOPIC_ARNS" required:"true"`
	MySQLHost          string `long:"mysql-host" env:"MYSQL_HOST" required:"true"`
	MySQLUser          string `long:"mysql-user" env:"MYSQL_USER" required:"true"`
	MySQLSMSecret      string `long:"mysql-secretsmanager-secret" env:"MYSQL_SECRETSMANAGER_SECRET" required:"true"`
	MySQLDatabase      string `long:"mysql-database" env:"MYSQL_DATABASE" required:"true"`
	FleetEnv           string `long:"fleet-environment" env:"FLEET_ENV" required:"true"`
	AWSRegion          string `long:"aws-region" env:"AWS_REGION" required:"true"`
	CronDelayTolerance string `long:"cron-delay-tolerance" env:"CRON_DELAY_TOLERANCE" default:"2h"`
}

var options = OptionsStruct{}

func sendSNSMessage(msg string, sess *session.Session) {
	log.Printf("Sending SNS Message")
	fullMsg := fmt.Sprintf("Environment: %s\nMessage: %s", options.FleetEnv, msg)
	svc := sns.New(sess)
	for _, SNSTopicArn := range strings.Split(options.SNSTopicArns, ",") {
		log.Printf("Sending '%s' to '%s'", fullMsg, SNSTopicArn)
		result, err := svc.Publish(&sns.PublishInput{
			Message:  &fullMsg,
			TopicArn: &SNSTopicArn,
		})
		if err != nil {
			log.Printf(err.Error())
		}
		log.Printf(result.GoString())
	}
}

func checkDB(sess *session.Session) (err error) {
	secretCache, err := secretcache.New()
	if err != nil {
		log.Printf(err.Error())
		sendSNSMessage("Unable to initialise SecretsManager helper.  Cron status is unknown.", sess)
		return err
	}

	secretCache.Client = secretsmanager.New(sess)

	MySQLPassword, err := secretCache.GetSecretString(options.MySQLSMSecret)
	if err != nil {
		log.Printf(err.Error())
		sendSNSMessage("Unable to retrieve SecretsManager secret.  Cron status is unknown.", sess)
		return err
	}

	cfg := mysql.Config{
		User:                 options.MySQLUser,
		Passwd:               MySQLPassword,
		Net:                  "tcp",
		Addr:                 options.MySQLHost,
		DBName:               options.MySQLDatabase,
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	defer db.Close()
	if err != nil {
		log.Printf(err.Error())
		sendSNSMessage("Unable to connect to database. Cron status unknown.", sess)
		return err
	}
	if err = db.Ping(); err != nil {
		log.Printf(err.Error())
		sendSNSMessage("Unable to connect to database. Cron status unknown.", sess)
		return err
	}

	log.Printf("Connected to database!")

	type CronStatsRow struct {
		name       string
		status     string
		updated_at time.Time
	}

	rows, err := db.Query("SELECT b.name,IFNULL(status, 'missing cron'),IFNULL(updated_at, FROM_UNIXTIME(0)) AS updated_at FROM (SELECT 'vulnerabilities' AS name UNION ALL SELECT 'cleanups_then_aggregation') b LEFT JOIN (SELECT name, status, updated_at FROM cron_stats WHERE id IN (SELECT MAX(id) FROM cron_stats WHERE status = 'completed' GROUP BY name)) a ON a.name = b.name;")
	defer rows.Close()
	if err != nil {
		log.Printf(err.Error())
		sendSNSMessage("Unable to SELECT cron_stats table.  Unable to continue.", sess)
		return err
	}
	cronDelayDuration, err := time.ParseDuration(options.CronDelayTolerance)
	if err != nil {
		log.Printf(err.Error())
		sendSNSMessage("Unable to parse cron-delay-tolerance. Check lambda settings.", sess)
		return err
	}
	cronAlertTimestamp := time.Now().Add(-1 * cronDelayDuration)
	for rows.Next() {
		var row CronStatsRow
		if err := rows.Scan(&row.name, &row.status, &row.updated_at); err != nil {
			log.Printf(err.Error())
			sendSNSMessage("Error scanning row in cron_stats table.  Unable to continue.", sess)
			return err
		}
		log.Printf("Row %s last updated at %s", row.name, row.updated_at.String())
		if row.updated_at.Before(cronAlertTimestamp) {
			log.Printf("*** %s hasn't updated in more than %s, alerting! (status %s)", options.CronDelayTolerance, row.name, row.status)
			// Fire on the first match and return.  We only need to alert that the crons need looked at, not each cron.
			sendSNSMessage(fmt.Sprintf("Fleet cron '%s' hasn't updated in more than %s. Last status was '%s' at %s.", row.name, options.CronDelayTolerance, row.status, row.updated_at.String()), sess)
			return nil
		}
	}

	return nil
}

func handler(ctx context.Context, name NullEvent) error {
	sess := session.Must(session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Config: aws.Config{
				Region: &options.AWSRegion,
			},
		},
	))

	checkDB(sess)
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

	// When running from Lambda, this should be read from the environment.
	if options.LambdaRuntimeAPI != "" {
		log.Printf("Starting Lambda handler.")
		lambda.Start(handler)
	} else {
		log.Printf("Lambda execution environment not found.  Falling back to local execution.")
		if err = handler(context.Background(), NullEvent{}); err != nil {
			log.Fatal(err)
		}
	}
}
