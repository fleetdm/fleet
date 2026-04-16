package fleet

import (
	"encoding/json"
	"time"
)

// NotificationType is a stable identifier for a category of system notification.
//
// The type determines:
//   - Which producer emits the notification.
//   - How the frontend may choose to group or style it.
//   - The shape of Notification.Metadata (each type documents its own schema).
//
// Types are stable string constants so they can be persisted, filtered on,
// and referenced across languages (Go, TypeScript).
type NotificationType string

const (
	// NotificationTypeAPNsCertExpiring — Apple Push Notification service cert
	// is within the 30-day expiration warning window.
	NotificationTypeAPNsCertExpiring NotificationType = "apns_cert_expiring"
	// NotificationTypeAPNsCertExpired — APNs cert has expired; MDM enrollments
	// will fail until a new cert is uploaded.
	NotificationTypeAPNsCertExpired NotificationType = "apns_cert_expired"
	// NotificationTypeABMTokenExpiring — Apple Business Manager server token
	// is within the 30-day warning window.
	NotificationTypeABMTokenExpiring NotificationType = "abm_token_expiring"
	// NotificationTypeABMTokenExpired — ABM server token has expired.
	NotificationTypeABMTokenExpired NotificationType = "abm_token_expired"
	// NotificationTypeABMTermsExpired — Apple Business Manager terms and
	// conditions need to be re-accepted.
	NotificationTypeABMTermsExpired NotificationType = "abm_terms_expired"
	// NotificationTypeVPPTokenExpiring — Volume Purchase Program token is
	// within the 30-day warning window.
	NotificationTypeVPPTokenExpiring NotificationType = "vpp_token_expiring"
	// NotificationTypeVPPTokenExpired — VPP token has expired.
	NotificationTypeVPPTokenExpired NotificationType = "vpp_token_expired"
	// NotificationTypeAndroidEnterpriseDeleted — Android Enterprise binding
	// was deleted out-of-band (detected via 404 from Android Enterprise API).
	NotificationTypeAndroidEnterpriseDeleted NotificationType = "android_enterprise_deleted"
	// NotificationTypeLicenseExpiring — premium Fleet license is within 30
	// days of expiring.
	NotificationTypeLicenseExpiring NotificationType = "license_expiring"
	// NotificationTypeLicenseExpired — premium license has expired; Fleet
	// continues to run but premium features may be affected.
	NotificationTypeLicenseExpired NotificationType = "license_expired"
)

// NotificationSeverity mirrors the three levels used by the frontend
// InfoBanner / flash-message components so the UI can pick colors and icons
// from a single known set.
type NotificationSeverity string

const (
	NotificationSeverityError   NotificationSeverity = "error"
	NotificationSeverityWarning NotificationSeverity = "warning"
	NotificationSeverityInfo    NotificationSeverity = "info"
)

// NotificationAudience controls which roles see a notification. Only "admin"
// is emitted today; the column exists so we can broaden audiences without a
// migration (see design doc: notifications are admin-only for v1).
type NotificationAudience string

const (
	NotificationAudienceAdmin NotificationAudience = "admin"
)

// Notification is a single system-generated event an admin may want to see or
// act on. It is the canonical row in the notifications table.
//
// Notifications are upserted by DedupeKey: a producer running on a cron should
// emit the same DedupeKey for the same condition so it updates in place
// instead of creating duplicates. When the underlying condition clears, the
// producer sets ResolvedAt — the row stays in history but is hidden from
// active lists.
//
// Per-user dismissal / read state lives in UserNotificationState, not here,
// so the same notification can be shown to multiple admins independently.
type Notification struct {
	ID         uint                 `json:"id" db:"id"`
	Type       NotificationType     `json:"type" db:"type"`
	Severity   NotificationSeverity `json:"severity" db:"severity"`
	Title      string               `json:"title" db:"title"`
	Body       string               `json:"body" db:"body"`
	CTAURL     *string              `json:"cta_url,omitempty" db:"cta_url"`
	CTALabel   *string              `json:"cta_label,omitempty" db:"cta_label"`
	Metadata   json.RawMessage      `json:"metadata,omitempty" db:"metadata"`
	DedupeKey  string               `json:"-" db:"dedupe_key"`
	Audience   NotificationAudience `json:"-" db:"audience"`
	ResolvedAt *time.Time           `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt  time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at" db:"updated_at"`

	// Per-user state fields — populated by ListNotificationsForUser via JOIN,
	// never read from / written to the notifications table directly.
	ReadAt      *time.Time `json:"read_at,omitempty" db:"read_at"`
	DismissedAt *time.Time `json:"dismissed_at,omitempty" db:"dismissed_at"`
}

// AuthzType identifies Notification to the OPA policy. All notifications are
// admin-only for v1; policy.rego enforces that.
func (n *Notification) AuthzType() string {
	return "notification"
}

// NotificationUpsert is the payload producers pass to
// Datastore.UpsertNotification. Only fields a producer controls appear here —
// ID, timestamps, and per-user state are managed by the datastore.
type NotificationUpsert struct {
	Type      NotificationType
	Severity  NotificationSeverity
	Title     string
	Body      string
	CTAURL    *string
	CTALabel  *string
	Metadata  json.RawMessage
	DedupeKey string
	Audience  NotificationAudience
}

// UserNotificationState is the per-user side-table row tracking whether a
// given user has read or dismissed a notification.
type UserNotificationState struct {
	UserID         uint       `json:"user_id" db:"user_id"`
	NotificationID uint       `json:"notification_id" db:"notification_id"`
	ReadAt         *time.Time `json:"read_at" db:"read_at"`
	DismissedAt    *time.Time `json:"dismissed_at" db:"dismissed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// NotificationListFilter describes how ListNotificationsForUser should filter
// the result set. The zero value returns active, non-dismissed notifications
// for the user — the common case for the profile-dropdown modal.
type NotificationListFilter struct {
	// IncludeDismissed when true returns rows the user has dismissed.
	IncludeDismissed bool
	// IncludeResolved when true returns rows whose underlying condition has
	// been resolved by the producer. Used by the full settings page.
	IncludeResolved bool
}
