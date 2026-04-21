package fleet

////////////////////////////////////////////////////////////////////////////////
// List notifications
////////////////////////////////////////////////////////////////////////////////

type ListNotificationsRequest struct {
	IncludeDismissed bool `query:"include_dismissed,optional"`
	IncludeResolved  bool `query:"include_resolved,optional"`
}

type ListNotificationsResponse struct {
	Notifications []*Notification `json:"notifications"`
	UnreadCount   int             `json:"unread_count"`
	Err           error           `json:"error,omitempty"`
}

func (r ListNotificationsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Notification summary (cheap — used to drive the profile-avatar badge)
////////////////////////////////////////////////////////////////////////////////

type NotificationSummaryRequest struct{}

type NotificationSummaryResponse struct {
	UnreadCount int   `json:"unread_count"`
	ActiveCount int   `json:"active_count"`
	Err         error `json:"error,omitempty"`
}

func (r NotificationSummaryResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Dismiss notification
////////////////////////////////////////////////////////////////////////////////

type DismissNotificationRequest struct {
	ID uint `url:"id"`
}

type DismissNotificationResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DismissNotificationResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Restore notification (un-dismiss)
////////////////////////////////////////////////////////////////////////////////

type RestoreNotificationRequest struct {
	ID uint `url:"id"`
}

type RestoreNotificationResponse struct {
	Err error `json:"error,omitempty"`
}

func (r RestoreNotificationResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Mark notification read
////////////////////////////////////////////////////////////////////////////////

type MarkNotificationReadRequest struct {
	ID uint `url:"id"`
}

type MarkNotificationReadResponse struct {
	Err error `json:"error,omitempty"`
}

func (r MarkNotificationReadResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Mark all notifications read
////////////////////////////////////////////////////////////////////////////////

type MarkAllNotificationsReadRequest struct{}

type MarkAllNotificationsReadResponse struct {
	Err error `json:"error,omitempty"`
}

func (r MarkAllNotificationsReadResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Demo: create a random test notification (admin-only, for demo/debug)
////////////////////////////////////////////////////////////////////////////////

type CreateDemoNotificationRequest struct{}

type CreateDemoNotificationResponse struct {
	Notification *Notification `json:"notification"`
	Err          error         `json:"error,omitempty"`
}

func (r CreateDemoNotificationResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Per-user notification preferences
////////////////////////////////////////////////////////////////////////////////

type ListNotificationPreferencesRequest struct{}

type ListNotificationPreferencesResponse struct {
	Preferences []UserNotificationPreference `json:"preferences"`
	Err         error                        `json:"error,omitempty"`
}

func (r ListNotificationPreferencesResponse) Error() error { return r.Err }

type UpdateNotificationPreferencesRequest struct {
	Preferences []UserNotificationPreference `json:"preferences"`
}

type UpdateNotificationPreferencesResponse struct {
	Preferences []UserNotificationPreference `json:"preferences"`
	Err         error                        `json:"error,omitempty"`
}

func (r UpdateNotificationPreferencesResponse) Error() error { return r.Err }
