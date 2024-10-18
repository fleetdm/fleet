package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Move all events with eventTitle from the primary calendar of the user to the new time.
// Only events in the future relative to the new event time are moved. In other words, if the current event time is in the past, it is not moved.
// Example: go run move-events.go --users john@example.com,jane@example.com --datetime 2024-04-01T10:00:00Z

var (
	serviceEmail = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL")
	privateKey   = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY")
)

const (
	eventTitle = "💻🚫 Scheduled maintenance"
)

func main() {
	if serviceEmail == "" || privateKey == "" {
		log.Fatal("FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL and FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY must be set")
	}
	// Strip newlines from private key
	privateKey = strings.ReplaceAll(privateKey, "\\n", "\n")
	userEmails := flag.String("users", "", "Comma-separated list of user emails to impersonate")
	dateTimeStr := flag.String("datetime", "", "Event time in "+time.RFC3339+" format")
	flag.Parse()
	if *userEmails == "" {
		log.Fatal("--users are required")
	}
	if *dateTimeStr == "" {
		log.Fatal("--datetime is required")
	}
	dateTime, err := time.Parse(time.RFC3339, *dateTimeStr)
	if err != nil {
		log.Fatalf("Unable to parse datetime: %v", err)
	}
	dateTimeEndStr := dateTime.Add(30 * time.Minute).Format(time.RFC3339)
	userEmailList := strings.Split(*userEmails, ",")
	if len(userEmailList) == 0 {
		log.Fatal("No user emails provided")
	}

	ctx := context.Background()

	var wg sync.WaitGroup

	for _, userEmail := range userEmailList {
		wg.Add(1)
		go func(userEmail string) {
			defer wg.Done()
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

			numberMoved := 0
			var maxResults int64 = 1000
			pageToken := ""
			now := time.Now()
			for {
				list, err := withRetry(
					func() (any, error) {
						return service.Events.List("primary").EventTypes("default").
							MaxResults(maxResults).
							OrderBy("startTime").
							SingleEvents(true).
							ShowDeleted(false).
							TimeMin(dateTimeEndStr).
							Q(eventTitle).
							PageToken(pageToken).
							Do()
					},
				)

				if err != nil {
					log.Fatalf("Unable to retrieve list of events: %v", err)
				}
				if len(list.(*calendar.Events).Items) == 0 {
					break
				}
				foundNewEvents := false
				for _, item := range list.(*calendar.Events).Items {
					created, err := time.Parse(time.RFC3339, item.Created)
					if err != nil {
						log.Fatalf("Unable to parse event created time: %v", err)
					}
					if created.After(now) {
						// Found events created after we started moving events, so we should stop
						foundNewEvents = true
						continue // Skip this event but finish the loop to make sure we don't miss something
					}
					if item.Summary == eventTitle {
						item.Start.DateTime = dateTime.Format(time.RFC3339)
						item.End.DateTime = dateTime.Add(30 * time.Minute).Format(time.RFC3339)
						_, err := withRetry(
							func() (any, error) {
								return service.Events.Update("primary", item.Id, item).Do()
							},
						)
						if err != nil {
							log.Fatalf("Unable to update event: %v", err)
						}
						numberMoved++
						if numberMoved%10 == 0 {
							log.Printf("Moved %d events for %s", numberMoved, userEmail)
						}

					}
				}
				pageToken = list.(*calendar.Events).NextPageToken
				if pageToken == "" || foundNewEvents {
					break
				}
			}
			log.Printf("DONE. Moved total %d events for %s", numberMoved, userEmail)
		}(userEmail)
	}

	// Wait for all goroutines to finish
	wg.Wait()

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
