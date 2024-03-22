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
)

// Delete all events with eventTitle from the primary calendar of the user.

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
	userEmail := flag.String("user", "", "User email to impersonate")
	flag.Parse()
	if *userEmail == "" {
		log.Fatal("--user is required")
	}

	ctx := context.Background()
	conf := &jwt.Config{
		Email: serviceEmail,
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar.events", "https://www.googleapis.com/auth/calendar.settings.readonly",
		},
		PrivateKey: []byte(privateKey),
		TokenURL:   google.JWTTokenURL,
		Subject:    *userEmail,
	}
	client := conf.Client(ctx)
	// Create a new calendar service
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create Calendar service: %v", err)
	}
	numberDeleted := 0
	for {
		list, err := service.Events.List("primary").EventTypes("default").MaxResults(1000).OrderBy("startTime").SingleEvents(true).ShowDeleted(false).Q(eventTitle).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve list of events: %v", err)
		}
		if len(list.Items) == 0 {
			break
		}
		for _, item := range list.Items {
			if item.Summary == eventTitle {
				err = service.Events.Delete("primary", item.Id).Do()
				if err != nil {
					log.Fatalf("Unable to delete event: %v", err)
				}
				numberDeleted++
				if numberDeleted%10 == 0 {
					log.Printf("Deleted %d events", numberDeleted)
				}
			}
		}
	}
	log.Printf("DONE. Deleted %d events total", numberDeleted)
}
