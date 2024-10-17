package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Stop watching the channel with the given ID. This command only accepts one user.
// Reference: https://developers.google.com/calendar/api/v3/reference/channels/stop

// Example: go run stop-channel.go --users john@example.com --channel-id 55ebefd7-4271-4295-a80a-97f4dcb01d93 --resource-id Io5ygBoEZ-FmQus7ziNrS_Jjcz4

var (
	serviceEmail = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL")
	privateKey   = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY")
)

func main() {
	if serviceEmail == "" || privateKey == "" {
		log.Fatal("FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL and FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY must be set")
	}
	// Strip newlines from private key
	privateKey = strings.ReplaceAll(privateKey, "\\n", "\n")
	userEmails := flag.String("users", "", "Comma-separated list of user emails to impersonate")
	channelIDStr := flag.String("channel-id", "", "Channel ID")
	resourceIDStr := flag.String("resource-id", "", "Resource ID")
	flag.Parse()
	if *userEmails == "" {
		log.Fatal("--users are required")
	}
	if *channelIDStr == "" {
		log.Fatal("--channel-id is required")
	}
	if *resourceIDStr == "" {
		log.Fatal("--resource-id is required")
	}
	userEmailList := strings.Split(*userEmails, ",")
	if len(userEmailList) == 0 {
		log.Fatal("No user emails provided")
	}
	if len(userEmailList) > 1 {
		log.Fatal("Only one user email is allowed")
	}

	ctx := context.Background()

	userEmail := userEmailList[0]
	conf := &jwt.Config{
		Email: serviceEmail,
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar.events", "https://www.googleapis.com/auth/calendar.settings.readonly",
		},
		PrivateKey: []byte(privateKey),
		TokenURL:   google.JWTTokenURL,
		Subject:    userEmail,
	}
	client := conf.Client(ctx)
	// Create a new calendar service
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Calendar service: %v", err)
	}

	_, err = withRetry(
		func() (any, error) {
			return nil, service.Channels.Stop(&calendar.Channel{
				Id:         *channelIDStr,
				ResourceId: *resourceIDStr,
			}).Do()
		},
	)

	if err != nil {
		log.Fatalf("Unable to stop watching channel: %v", err)
	}
	log.Printf("DONE. Stopped watching channel resource for %s", userEmail)

}

func withRetry(fn func() (any, error)) (any, error) {
	retryStrategy := backoff.NewExponentialBackOff()
	retryStrategy.MaxElapsedTime = 60 * time.Minute
	var result any
	err := backoff.Retry(
		func() error {
			var err error
			result, err = fn()
			if err != nil {
				if isRateLimited(err) {
					return err
				}
				return backoff.Permanent(err)
			}
			return nil
		}, retryStrategy,
	)
	return result, err
}

func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	var ae *googleapi.Error
	ok := errors.As(err, &ae)
	return ok && (ae.Code == http.StatusTooManyRequests ||
		(ae.Code == http.StatusForbidden &&
			(ae.Message == "Rate Limit Exceeded" || ae.Message == "User Rate Limit Exceeded" || ae.Message == "Calendar usage limits exceeded." || strings.HasPrefix(
				ae.Message, "Quota exceeded",
			))))
}
