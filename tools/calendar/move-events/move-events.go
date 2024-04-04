package main

import (
	"context"
	"flag"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// Move all events with eventTitle from the primary calendar of the user to the new time.
// Only events in the future relative to the new event time are moved. In other words, if the current event time is in the past, it is not moved.

var (
	serviceEmail = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL")
	privateKey   = os.Getenv("FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY")
)

const (
	eventTitle = "💻🚫Downtime"
)

func main() {
	if serviceEmail == "" || privateKey == "" {
		log.Fatal("FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL and FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY must be set")
	}
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
			for {
				list, err := service.Events.List("primary").EventTypes("default").
					MaxResults(1000).
					OrderBy("startTime").
					SingleEvents(true).
					ShowDeleted(false).
					TimeMin(dateTimeEndStr).
					Q(eventTitle).
					Do()
				if err != nil {
					log.Fatalf("Unable to retrieve list of events: %v", err)
				}
				if len(list.Items) == 0 {
					break
				}
				for _, item := range list.Items {
					if item.Summary == eventTitle {
						item.Start.DateTime = dateTime.Format(time.RFC3339)
						item.End.DateTime = dateTime.Add(30 * time.Minute).Format(time.RFC3339)
						_, err := service.Events.Update("primary", item.Id, item).Do()
						if err != nil {
							log.Fatalf("Unable to update event: %v", err)
						}
						numberMoved++
					}
				}
			}
			log.Printf("Moved %d events for %s", numberMoved, userEmail)
		}(userEmail)
	}

	// Wait for all goroutines to finish
	wg.Wait()

}
