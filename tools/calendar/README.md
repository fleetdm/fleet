# Helper methods for Google calendar

To delete all downtime events from a Google Calendar, use `delete-events/delete-events.go`

To move all downtime events from multiple Google Calendars to a specific time, use `move-events/move-events.go`

To use the helper scripts, you must set `FLEET_TEST_GOOGLE_CALENDAR_SERVICE_EMAIL` and `FLEET_TEST_GOOGLE_CALENDAR_PRIVATE_KEY` environment variables. The email is the `client_email` from JSON key file. The private key also comes from JSON key file for the service account, and starts with `-----BEGIN PRIVATE KEY-----`.

# Calendar server for load testing

Test calendar server that provides a REST API for managing events.
Since we may not have access to a real calendar server (such as Google Calendar API), this server will be used to test the calendar feature during load testing.

Start the server like:
```shell
go run calendar.go --port 8083 --db ./calendar.db
```

The server uses a SQLite database to store events. This database can be modified during testing.

On the fleet server, configure Google Calendar API key where `client_email` is the specified value and the `private_key` is the base URL of the calendar server:
```json
{
    "client_email": "calendar-load@example.com",
    "private_key": "http://localhost:8083"
}
```

## Useful tricks

To update all the events in SQLite database to start at the current time, do SQL query:
```sql
UPDATE events SET start = unixepoch('now'), end = unixepoch('now', '+30 minutes');
```
