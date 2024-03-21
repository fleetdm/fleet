# Calendar server for load testing

Test calendar server that provides a REST API for managing events.
Since we may not have access to a real calendar server (such as Google Calendar API), this server will be used to test the calendar feature during load testing.

Start the server like:
```zsh
go run calendar.go --port 8083 --db ./calendar.db
```

The server uses a SQLite database to store events. This database can be modified during testing.

On the fleet server, configure Google Calendar API key where `client_email` is the specified value and the `private_key` is the base URL of the calendar server:
```
{
    "client_email": "calendar-load@example.com",
    "private_key": "http://localhost:8083"
}
```
