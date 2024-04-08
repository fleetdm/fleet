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

func (ds *Datastore) CreateOrUpdateCalendarEvent(
	ctx context.Context,
	email string,
	startTime time.Time,
	endTime time.Time,
	data []byte,
	hostID uint,
	webhookStatus fleet.CalendarWebhookStatus,
) (*fleet.CalendarEvent, error) {
	var id int64
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		const calendarEventsQuery = `
			INSERT INTO calendar_events (
				email,
				start_time,
				end_time,
				event
			) VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				start_time = VALUES(start_time),
				end_time = VALUES(end_time),
				event = VALUES(event),
				updated_at = CURRENT_TIMESTAMP;
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

		if insertOnDuplicateDidInsert(result) {
			id, _ = result.LastInsertId()
		} else {
			stmt := `SELECT id FROM calendar_events WHERE email = ?`
			if err := sqlx.GetContext(ctx, tx, &id, stmt, email); err != nil {
				return ctxerr.Wrap(ctx, err, "calendar event id")
			}
		}

		const hostCalendarEventsQuery = `
			INSERT INTO host_calendar_events (
				host_id,
				calendar_event_id,
				webhook_status
			) VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE
				webhook_status = VALUES(webhook_status),
				calendar_event_id = VALUES(calendar_event_id);
		`
		result, err = tx.ExecContext(
			ctx,
			hostCalendarEventsQuery,
			hostID,
			id,
			webhookStatus,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert host calendar event")
		}
		return nil
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	calendarEvent, err := getCalendarEventByID(ctx, ds.writer(ctx), uint(id))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get created calendar event by id")
	}
	return calendarEvent, nil
}

func getCalendarEventByID(ctx context.Context, q sqlx.QueryerContext, id uint) (*fleet.CalendarEvent, error) {
	const calendarEventsQuery = `
		SELECT * FROM calendar_events WHERE id = ?;
	`
	var calendarEvent fleet.CalendarEvent
	err := sqlx.GetContext(ctx, q, &calendarEvent, calendarEventsQuery, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("CalendarEvent").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "get calendar event")
	}
	return &calendarEvent, nil
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
			event = ?,
			updated_at = CURRENT_TIMESTAMP
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

func (ds *Datastore) GetHostCalendarEventByEmail(ctx context.Context, email string) (*fleet.HostCalendarEvent, *fleet.CalendarEvent, error) {
	const calendarEventsQuery = `
		SELECT * FROM calendar_events WHERE email = ?
	`
	var calendarEvent fleet.CalendarEvent
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &calendarEvent, calendarEventsQuery, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ctxerr.Wrap(ctx, notFound("CalendarEvent").WithMessage(fmt.Sprintf("email: %s", email)))
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "get calendar event")
	}
	const hostCalendarEventsQuery = `
		SELECT * FROM host_calendar_events WHERE calendar_event_id = ?
	`
	var hostCalendarEvent fleet.HostCalendarEvent
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostCalendarEvent, hostCalendarEventsQuery, calendarEvent.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ctxerr.Wrap(ctx, notFound("HostCalendarEvent").WithID(calendarEvent.ID))
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "get host calendar event")
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

func (ds *Datastore) ListCalendarEvents(ctx context.Context, teamID *uint) ([]*fleet.CalendarEvent, error) {
	calendarEventsQuery := `
		SELECT ce.* FROM calendar_events ce
	`

	var args []interface{}
	if teamID != nil {
		calendarEventsQuery += ` JOIN host_calendar_events hce ON ce.id=hce.calendar_event_id
								 JOIN hosts h ON h.id=hce.host_id WHERE h.team_id = ?`
		args = append(args, *teamID)
	}

	var calendarEvents []*fleet.CalendarEvent
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &calendarEvents, calendarEventsQuery, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get all calendar events")
	}
	return calendarEvents, nil
}

func (ds *Datastore) ListOutOfDateCalendarEvents(ctx context.Context, t time.Time) ([]*fleet.CalendarEvent, error) {
	calendarEventsQuery := `
		SELECT ce.* FROM calendar_events ce WHERE updated_at < ?
	`
	var calendarEvents []*fleet.CalendarEvent
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &calendarEvents, calendarEventsQuery, t); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get all calendar events")
	}
	return calendarEvents, nil
}
