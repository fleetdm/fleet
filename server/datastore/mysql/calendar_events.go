package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewCalendarEvent(
	ctx context.Context,
	email string,
	startTime time.Time,
	endTime time.Time,
	data []byte,
	hostID uint,
) (*fleet.CalendarEvent, error) {
	var calendarEvent *fleet.CalendarEvent
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		const calendarEventsQuery = `
			INSERT INTO calendar_events (
				email,
				start_time,
				end_time,
				event
			) VALUES (?, ?, ?, ?);
		`
		result, err := tx.ExecContext(
			ctx,
			calendarEventsQuery,
			email,
			startTime,
			endTime,
			data,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert calendar event")
		}

		id, _ := result.LastInsertId()
		calendarEvent = &fleet.CalendarEvent{
			ID:        uint(id),
			Email:     email,
			StartTime: startTime,
			EndTime:   endTime,
			Data:      data,
		}

		const hostCalendarEventsQuery = `
			INSERT INTO host_calendar_events (
				host_id,
				calendar_event_id,
				webhook_status
			) VALUES (?, ?, ?);
		`
		result, err = tx.ExecContext(
			ctx,
			hostCalendarEventsQuery,
			hostID,
			calendarEvent.ID,
			fleet.CalendarWebhookStatusPending,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert host calendar event")
		}
		return nil
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return calendarEvent, nil
}

func (ds *Datastore) GetCalendarEvent(ctx context.Context, email string) (*fleet.CalendarEvent, error) {
	const calendarEventsQuery = `
		SELECT * FROM calendar_events WHERE email = ?;
	`
	var calendarEvent fleet.CalendarEvent
	err := sqlx.GetContext(ctx, ds.reader(ctx), &calendarEvent, calendarEventsQuery, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("CalendarEvent").WithMessage(fmt.Sprintf("email: %s", email)))
		}
		return nil, ctxerr.Wrap(ctx, err, "get calendar event")
	}
	return &calendarEvent, nil
}

func (ds *Datastore) UpdateCalendarEvent(ctx context.Context, calendarEventID uint, startTime time.Time, endTime time.Time, data []byte) error {
	const calendarEventsQuery = `
		UPDATE calendar_events SET
			start_time = ?,
			end_time = ?,
			event = ?
		WHERE id = ?;
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, calendarEventsQuery, startTime, endTime, data, calendarEventID); err != nil {
		return ctxerr.Wrap(ctx, err, "update calendar event")
	}
	return nil
}

func (ds *Datastore) DeleteCalendarEvent(ctx context.Context, calendarEventID uint) error {
	const calendarEventsQuery = `
		DELETE FROM calendar_events WHERE id = ?;
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, calendarEventsQuery, calendarEventID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete calendar event")
	}
	return nil
}

func (ds *Datastore) GetHostCalendarEvent(ctx context.Context, hostID uint) (*fleet.HostCalendarEvent, *fleet.CalendarEvent, error) {
	const hostCalendarEventsQuery = `
		SELECT * FROM host_calendar_events WHERE host_id = ?
	`
	var hostCalendarEvent fleet.HostCalendarEvent
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostCalendarEvent, hostCalendarEventsQuery, hostID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ctxerr.Wrap(ctx, notFound("HostCalendarEvent").WithMessage(fmt.Sprintf("host_id: %d", hostID)))
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "get host calendar event")
	}
	const calendarEventsQuery = `
		SELECT * FROM calendar_events WHERE id = ?
	`
	var calendarEvent fleet.CalendarEvent
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &calendarEvent, calendarEventsQuery, hostCalendarEvent.CalendarEventID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ctxerr.Wrap(ctx, notFound("CalendarEvent").WithID(hostCalendarEvent.CalendarEventID))
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "get calendar event")
	}
	return &hostCalendarEvent, &calendarEvent, nil
}

func (ds *Datastore) UpdateHostCalendarWebhookStatus(ctx context.Context, hostID uint, status fleet.CalendarWebhookStatus) error {
	const calendarEventsQuery = `
		UPDATE host_calendar_events SET
			webhook_status = ?
		WHERE host_id = ?;
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, calendarEventsQuery, status, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "update host calendar event webhook status")
	}
	return nil
}
