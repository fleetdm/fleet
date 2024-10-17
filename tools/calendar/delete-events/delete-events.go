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

// Delete all events with eventTitle from the primary calendar of the specified users.
// Example: go run delete-events.go --users john@example.com,jane@example.com

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
	flag.Parse()
	if *userEmails == "" {
		log.Fatal("--users are required")
	}
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
			numberDeleted := 0
			var maxResults int64 = 1000
			pageToken := ""
			now := time.Now()
			for {
				list, err := withRetry(
					func() (any, error) {
						return service.Events.List("primary").
							EventTypes("default").
							MaxResults(maxResults).
							OrderBy("startTime").
							SingleEvents(true).
							ShowDeleted(false).
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
						// Found events created after we started deleting events, so we should stop
						foundNewEvents = true
						continue // Skip this event but finish the loop to make sure we don't miss something
					}
					if item.Summary == eventTitle {
						_, err := withRetry(
							func() (any, error) {
								return nil, service.Events.Delete("primary", item.Id).Do()
							},
						)
						if err != nil {
							log.Fatalf("Unable to delete event: %v", err)
						}
						numberDeleted++
						if numberDeleted%10 == 0 {
							log.Printf("Deleted %d events for %s", numberDeleted, userEmail)
						}
					}
				}
				pageToken = list.(*calendar.Events).NextPageToken
				if pageToken == "" || foundNewEvents {
					break
				}
			}
			log.Printf("DONE. Deleted %d events total for %s", numberDeleted, userEmail)
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
