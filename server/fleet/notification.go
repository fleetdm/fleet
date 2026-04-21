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

// NotificationCategory groups related NotificationTypes so users can opt in or
// out of a whole class of notifications without us listing every type. A user
// who doesn't own vulnerabilities can mute the "vulnerabilities" category and
// still receive MDM / license notifications.
//
// The mapping from type → category lives in NotificationTypeCategory. Adding a
// new NotificationType requires adding it to that map so it is routed to the
// correct category and respects user preferences.
type NotificationCategory string

const (
	NotificationCategoryMDM             NotificationCategory = "mdm"
	NotificationCategoryLicense         NotificationCategory = "license"
	NotificationCategoryVulnerabilities NotificationCategory = "vulnerabilities"
	NotificationCategoryPolicies        NotificationCategory = "policies"
	NotificationCategorySoftware        NotificationCategory = "software"
	NotificationCategoryHosts           NotificationCategory = "hosts"
	NotificationCategoryIntegrations    NotificationCategory = "integrations"
	NotificationCategorySystem          NotificationCategory = "system"
	// NotificationCategoryAll is a sentinel that only makes sense on delivery
	// routes (Slack webhook configs, etc.) — it matches every real category.
	// It never appears on a notification row; CategoryForType never returns it.
	NotificationCategoryAll NotificationCategory = "all"
)

// AllNotificationCategories is the canonical list used by the prefs API and
// the My Account UI. The order is the order the categories are rendered.
var AllNotificationCategories = []NotificationCategory{
	NotificationCategoryMDM,
	NotificationCategoryLicense,
	NotificationCategoryVulnerabilities,
	NotificationCategoryPolicies,
	NotificationCategorySoftware,
	NotificationCategoryHosts,
	NotificationCategoryIntegrations,
	NotificationCategorySystem,
}

// NotificationTypeCategory maps every known NotificationType to the category
// it belongs to. Unknown types fall back to NotificationCategorySystem — this
// keeps an unregistered demo or partner type visible rather than silently
// dropping it, at the cost of classifying it generically.
var NotificationTypeCategory = map[NotificationType]NotificationCategory{
	NotificationTypeAPNsCertExpiring:         NotificationCategoryMDM,
	NotificationTypeAPNsCertExpired:          NotificationCategoryMDM,
	NotificationTypeABMTokenExpiring:         NotificationCategoryMDM,
	NotificationTypeABMTokenExpired:          NotificationCategoryMDM,
	NotificationTypeABMTermsExpired:          NotificationCategoryMDM,
	NotificationTypeVPPTokenExpiring:         NotificationCategoryMDM,
	NotificationTypeVPPTokenExpired:          NotificationCategoryMDM,
	NotificationTypeAndroidEnterpriseDeleted: NotificationCategoryMDM,
	NotificationTypeLicenseExpiring:          NotificationCategoryLicense,
	NotificationTypeLicenseExpired:           NotificationCategoryLicense,

	// Demo notification types — map here so the category-based preference
	// filter works for them in dev and demo environments.
	"demo_hosts_offline":     NotificationCategoryHosts,
	"demo_policy_failures":   NotificationCategoryPolicies,
	"demo_vuln_cisa_kev":     NotificationCategoryVulnerabilities,
	"demo_software_failures": NotificationCategorySoftware,
	"demo_fleet_update":      NotificationCategorySystem,
	"demo_seat_limit":        NotificationCategoryLicense,
	"demo_disk_encryption":   NotificationCategoryMDM,
	"demo_webhook_failing":   NotificationCategoryIntegrations,
}

// CategoryForType returns the category for a given NotificationType. Unknown
// types land in the "system" bucket; see NotificationTypeCategory for detail.
func CategoryForType(t NotificationType) NotificationCategory {
	if c, ok := NotificationTypeCategory[t]; ok {
		return c
	}
	return NotificationCategorySystem
}

// NotificationChannel identifies a delivery channel for a notification. Only
// in_app is read by Fleet today; the column exists so preferences for email
// and slack can land ahead of the actual delivery pipeline.
type NotificationChannel string

const (
	NotificationChannelInApp NotificationChannel = "in_app"
	NotificationChannelEmail NotificationChannel = "email"
	NotificationChannelSlack NotificationChannel = "slack"
)

// AllNotificationChannels is the canonical order the UI renders the channel
// toggles. Kept narrow for now — the prefs API only surfaces in_app until the
// delivery workers for email/slack land.
var AllNotificationChannels = []NotificationChannel{
	NotificationChannelInApp,
	NotificationChannelEmail,
	NotificationChannelSlack,
}

// UserNotificationPreference is one row of per-user opt-in state. Rows exist
// only for (user, category, channel) combinations the user has explicitly
// toggled away from the default. Absence of a row means "enabled" — users
// receive new notifications by default.
type UserNotificationPreference struct {
	UserID   uint                 `json:"-" db:"user_id"`
	Category NotificationCategory `json:"category" db:"category"`
	Channel  NotificationChannel  `json:"channel" db:"channel"`
	Enabled  bool                 `json:"enabled" db:"enabled"`
}

// NotificationDeliveryStatus tracks the lifecycle of a single fanout row in
// notification_deliveries. The cron worker transitions pending → sent|failed.
type NotificationDeliveryStatus string

const (
	NotificationDeliveryStatusPending NotificationDeliveryStatus = "pending"
	NotificationDeliveryStatusSent    NotificationDeliveryStatus = "sent"
	NotificationDeliveryStatusFailed  NotificationDeliveryStatus = "failed"
)

// NotificationDelivery is one row in the notification_deliveries table —
// the scheduled (or completed) fanout of a notification to a single
// destination (e.g. a Slack webhook URL).
type NotificationDelivery struct {
	ID             uint                       `db:"id"`
	NotificationID uint                       `db:"notification_id"`
	Channel        NotificationChannel        `db:"channel"`
	Target         string                     `db:"target"`
	Status         NotificationDeliveryStatus `db:"status"`
	Error          *string                    `db:"error"`
	AttemptedAt    *time.Time                 `db:"attempted_at"`
	CreatedAt      time.Time                  `db:"created_at"`
	UpdatedAt      time.Time                  `db:"updated_at"`
}

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
