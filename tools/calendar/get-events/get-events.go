package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
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

// Get all events with eventTitle from the primary calendar of the specified users.
// Example: go run delete-events.go --users john@example.com,jane@example.com

var (
	serviceEmail = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL")
	privateKey   = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY")
)

const (
	eventTitle = "💻🚫 Scheduled maintenance"
)

var regexMachineName = regexp.MustCompile(`your work computer (because there was no remaining availability )?\((?P<machine>.*)\)\.`)

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

	type summary struct {
		total       int
		totalByDate map[string]int
		duplicates  map[string]struct{}
	}
	summaryByUser := make(map[string]summary)

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
			var maxResults int64 = 1000
			pageToken := ""
			var total = 0
			var totalByDate = make(map[string]int)
			var machines = make(map[string]struct{})
			var duplicates = make(map[string]struct{})
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
				for _, item := range list.(*calendar.Events).Items {
					if item.Summary == eventTitle {
						created, err := time.Parse(time.RFC3339, item.Created)
						if err != nil {
							log.Fatalf("Unable to parse event created time: %v", err)
						}
						var startTime time.Time
						if item.Start != nil {
							startTime, err = time.Parse(time.RFC3339, item.Start.DateTime)
							if err != nil {
								log.Fatalf("Unable to parse event start time: %v", err)
							}
						}
						matches := regexMachineName.FindStringSubmatch(item.Description)
						machineName := "NOT_FOUND"
						if matches != nil {
							machineName = matches[regexMachineName.SubexpIndex("machine")]
							if _, ok := machines[machineName]; ok {
								duplicates[machineName] = struct{}{}
							}
							machines[machineName] = struct{}{}
						}
						total += 1
						dateStr := startTime.Format("2006-01-02")
						totalByDate[dateStr] += 1
						fmt.Printf("%s created_at:%s user:%s machine:%s\n", startTime.Format(time.RFC3339), created.Format(time.RFC3339),
							userEmail, machineName)
					}
				}
				pageToken = list.(*calendar.Events).NextPageToken
				if pageToken == "" || len(list.(*calendar.Events).Items) == 0 {
					summaryByUser[userEmail] = summary{total: total, totalByDate: totalByDate, duplicates: duplicates}
					break
				}
			}
		}(userEmail)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Printf("Summary:\n")
	for userEmail, s := range summaryByUser {
		fmt.Printf("User: %s, Total: %d\n", userEmail, s.total)
		for date, count := range s.totalByDate {
			fmt.Printf("User: %s, Date: %s, Count: %d\n", userEmail, date, count)
		}
		if len(s.duplicates) > 0 {
			dups := make([]string, 0, len(s.duplicates))
			for k := range s.duplicates {
				dups = append(dups, k)
			}
			fmt.Printf("User: %s, Machines with multiple events: %v\n", userEmail, dups)
		}
	}

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
