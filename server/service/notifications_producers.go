package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
)

// notificationExpiryWindow is how far in advance we surface an "expiring"
// notification before the actual expiration date. Matches the 30-day window
// the existing frontend banners use.
const notificationExpiryWindow = 30 * 24 * time.Hour

// runNotificationProducers evaluates every known notification-emitting
// condition and upserts or resolves notifications accordingly.
//
// This is called synchronously from ListNotifications and NotificationSummary
// so the badge / dropdown always reflects the most recent state. Each
// producer is isolated — an error from one does not prevent the others from
// running; errors are wrapped and returned as a single joined error so the
// caller can log and continue.
//
// Post-hackathon this should move to a dedicated cron schedule. The current
// synchronous approach is intentional for v1: simpler, always-fresh, and the
// small cost (5-10 cheap queries on a rarely-hit endpoint) is acceptable.
func (svc *Service) runNotificationProducers(ctx context.Context) {
	// Each producer logs and swallows its own error — we do not want one
	// producer failure to take down the notification list.
	type producer struct {
		name string
		run  func(context.Context) error
	}
	producers := []producer{
		{"license", svc.producerLicense},
		{"abm_terms", svc.producerABMTerms},
		{"abm_tokens", svc.producerABMTokens},
		{"vpp_tokens", svc.producerVPPTokens},
		{"apns_cert", svc.producerAPNsCert},
		{"android_enterprise", svc.producerAndroidEnterprise},
	}
	for _, p := range producers {
		if err := p.run(ctx); err != nil {
			svc.logger.WarnContext(ctx, "notification producer failed", "producer", p.name, "err", err)
		}
	}
}

// producerLicense emits license_expiring or license_expired notifications
// when the premium license is within 30 days of expiry, or past it.
// Resolves both keys otherwise (free tier or healthy license).
func (svc *Service) producerLicense(ctx context.Context) error {
	lic, err := svc.License(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get license")
	}
	// Free tier — nothing to warn about.
	if lic == nil || !lic.IsPremium() || lic.Expiration.IsZero() {
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeLicenseExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeLicenseExpired))
		return nil
	}

	now := time.Now()
	switch {
	case lic.Expiration.Before(now):
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeLicenseExpired,
			Severity:  fleet.NotificationSeverityError,
			Title:     "Premium license expired",
			Body:      fmt.Sprintf("Your Fleet Premium license expired on %s. Renew to keep premium features.", lic.Expiration.Format("2006-01-02")),
			CTAURL:    new("https://fleetdm.com/customers/register"),
			CTALabel:  new("Renew license"),
			Metadata:  mustJSON(map[string]any{"expiration": lic.Expiration}),
			DedupeKey: string(fleet.NotificationTypeLicenseExpired),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeLicenseExpiring))
		return err
	case lic.Expiration.Before(now.Add(notificationExpiryWindow)):
		days := int(time.Until(lic.Expiration).Hours() / 24)
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeLicenseExpiring,
			Severity:  fleet.NotificationSeverityWarning,
			Title:     "Premium license expires soon",
			Body:      fmt.Sprintf("Your Fleet Premium license expires in %d days (%s). Renew to avoid interruption.", days, lic.Expiration.Format("2006-01-02")),
			CTAURL:    new("https://fleetdm.com/customers/register"),
			CTALabel:  new("Renew license"),
			Metadata:  mustJSON(map[string]any{"expiration": lic.Expiration, "days_until": days}),
			DedupeKey: string(fleet.NotificationTypeLicenseExpiring),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeLicenseExpired))
		return err
	default:
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeLicenseExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeLicenseExpired))
		return nil
	}
}

// producerABMTerms emits abm_terms_expired when any ABM token has
// terms_expired set. The AppConfig exposes a convenience flag (set whenever
// any token has the flag) that we prefer over scanning every token.
func (svc *Service) producerABMTerms(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config for abm terms")
	}
	if appCfg.MDM.AppleBMTermsExpired {
		_, err := svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeABMTermsExpired,
			Severity:  fleet.NotificationSeverityError,
			Title:     "Apple Business Manager terms need renewal",
			Body:      "Your organization has not accepted the latest Apple Business Manager terms and conditions. Sign in to Apple Business Manager to review and accept them.",
			CTAURL:    new("https://business.apple.com/"),
			CTALabel:  new("Review terms"),
			DedupeKey: string(fleet.NotificationTypeABMTermsExpired),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		return err
	}
	_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeABMTermsExpired))
	return nil
}

// producerABMTokens emits abm_token_expiring / abm_token_expired based on the
// earliest renew date across all ABM tokens. A single notification is emitted
// covering the soonest-expiring token — the body names the org so admins know
// which one.
func (svc *Service) producerABMTokens(ctx context.Context) error {
	tokens, err := svc.ds.ListABMTokens(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list abm tokens")
	}
	if len(tokens) == 0 {
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeABMTokenExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeABMTokenExpired))
		return nil
	}

	var earliest *fleet.ABMToken
	for _, t := range tokens {
		if t.RenewAt.IsZero() {
			continue
		}
		if earliest == nil || t.RenewAt.Before(earliest.RenewAt) {
			earliest = t
		}
	}
	if earliest == nil {
		return nil
	}

	now := time.Now()
	cta := new("/settings/integrations/mdm/apple-business-manager")
	ctaLabel := new("Renew token")

	switch {
	case earliest.RenewAt.Before(now):
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeABMTokenExpired,
			Severity:  fleet.NotificationSeverityError,
			Title:     "Apple Business Manager token expired",
			Body:      fmt.Sprintf("The ABM token for %q expired on %s. Upload a new token to keep macOS/iOS/iPadOS auto-enrollment working.", earliest.OrganizationName, earliest.RenewAt.Format("2006-01-02")),
			CTAURL:    cta,
			CTALabel:  ctaLabel,
			Metadata:  mustJSON(map[string]any{"org": earliest.OrganizationName, "renew_at": earliest.RenewAt}),
			DedupeKey: string(fleet.NotificationTypeABMTokenExpired),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeABMTokenExpiring))
		return err
	case earliest.RenewAt.Before(now.Add(notificationExpiryWindow)):
		days := int(time.Until(earliest.RenewAt).Hours() / 24)
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeABMTokenExpiring,
			Severity:  fleet.NotificationSeverityWarning,
			Title:     "Apple Business Manager token expiring soon",
			Body:      fmt.Sprintf("The ABM token for %q expires in %d days. Upload a new token to avoid losing auto-enrollment.", earliest.OrganizationName, days),
			CTAURL:    cta,
			CTALabel:  ctaLabel,
			Metadata:  mustJSON(map[string]any{"org": earliest.OrganizationName, "renew_at": earliest.RenewAt, "days_until": days}),
			DedupeKey: string(fleet.NotificationTypeABMTokenExpiring),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeABMTokenExpired))
		return err
	default:
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeABMTokenExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeABMTokenExpired))
		return nil
	}
}

// producerVPPTokens mirrors producerABMTokens for Volume Purchase Program.
func (svc *Service) producerVPPTokens(ctx context.Context) error {
	tokens, err := svc.ds.ListVPPTokens(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list vpp tokens")
	}
	if len(tokens) == 0 {
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeVPPTokenExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeVPPTokenExpired))
		return nil
	}

	var earliest *fleet.VPPTokenDB
	for _, t := range tokens {
		if t.RenewDate.IsZero() {
			continue
		}
		if earliest == nil || t.RenewDate.Before(earliest.RenewDate) {
			earliest = t
		}
	}
	if earliest == nil {
		return nil
	}

	now := time.Now()
	cta := new("/settings/integrations/mdm/volume-purchasing-program")
	ctaLabel := new("Renew token")

	switch {
	case earliest.RenewDate.Before(now):
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeVPPTokenExpired,
			Severity:  fleet.NotificationSeverityError,
			Title:     "VPP token expired",
			Body:      fmt.Sprintf("The VPP token for %q expired on %s. Upload a new token to keep App Store app installs working.", earliest.OrgName, earliest.RenewDate.Format("2006-01-02")),
			CTAURL:    cta,
			CTALabel:  ctaLabel,
			Metadata:  mustJSON(map[string]any{"org": earliest.OrgName, "renew_at": earliest.RenewDate}),
			DedupeKey: string(fleet.NotificationTypeVPPTokenExpired),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeVPPTokenExpiring))
		return err
	case earliest.RenewDate.Before(now.Add(notificationExpiryWindow)):
		days := int(time.Until(earliest.RenewDate).Hours() / 24)
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeVPPTokenExpiring,
			Severity:  fleet.NotificationSeverityWarning,
			Title:     "VPP token expiring soon",
			Body:      fmt.Sprintf("The VPP token for %q expires in %d days. Upload a new token to avoid losing App Store app installs.", earliest.OrgName, days),
			CTAURL:    cta,
			CTALabel:  ctaLabel,
			Metadata:  mustJSON(map[string]any{"org": earliest.OrgName, "renew_at": earliest.RenewDate, "days_until": days}),
			DedupeKey: string(fleet.NotificationTypeVPPTokenExpiring),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeVPPTokenExpired))
		return err
	default:
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeVPPTokenExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeVPPTokenExpired))
		return nil
	}
}

// mustJSON marshals v and ignores errors — inputs here are always plain
// maps with serializable values, so any error is a programmer bug.
func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

// producerAPNsCert loads the APNs push certificate, parses its x509 NotAfter,
// and emits apns_cert_expiring or apns_cert_expired accordingly. When no cert
// is configured (ErrPartialResult / not-found) both keys are resolved.
func (svc *Service) producerAPNsCert(ctx context.Context) error {
	cert, err := assets.X509Cert(ctx, svc.ds, fleet.MDMAssetAPNSCert)
	if err != nil {
		// No cert configured — nothing to warn about.
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeAPNsCertExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeAPNsCertExpired))
		return nil //nolint:nilerr // expected when Apple MDM is not set up
	}

	now := time.Now()
	cta := new("/settings/integrations/mdm/apple")
	ctaLabel := new("Renew certificate")

	switch {
	case cert.NotAfter.Before(now):
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeAPNsCertExpired,
			Severity:  fleet.NotificationSeverityError,
			Title:     "APNs certificate expired",
			Body:      fmt.Sprintf("Your Apple Push Notification service certificate expired on %s. macOS/iOS/iPadOS MDM enrollments will fail until a new certificate is uploaded.", cert.NotAfter.Format("2006-01-02")),
			CTAURL:    cta,
			CTALabel:  ctaLabel,
			Metadata:  mustJSON(map[string]any{"not_after": cert.NotAfter, "common_name": cert.Subject.CommonName}),
			DedupeKey: string(fleet.NotificationTypeAPNsCertExpired),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeAPNsCertExpiring))
		return err
	case cert.NotAfter.Before(now.Add(notificationExpiryWindow)):
		days := int(time.Until(cert.NotAfter).Hours() / 24)
		_, err = svc.ds.UpsertNotification(ctx, fleet.NotificationUpsert{
			Type:      fleet.NotificationTypeAPNsCertExpiring,
			Severity:  fleet.NotificationSeverityWarning,
			Title:     "APNs certificate expiring soon",
			Body:      fmt.Sprintf("Your Apple Push Notification service certificate expires in %d days (%s). Renew it to avoid MDM enrollment failures.", days, cert.NotAfter.Format("2006-01-02")),
			CTAURL:    cta,
			CTALabel:  ctaLabel,
			Metadata:  mustJSON(map[string]any{"not_after": cert.NotAfter, "days_until": days, "common_name": cert.Subject.CommonName}),
			DedupeKey: string(fleet.NotificationTypeAPNsCertExpiring),
			Audience:  fleet.NotificationAudienceAdmin,
		})
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeAPNsCertExpired))
		return err
	default:
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeAPNsCertExpiring))
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeAPNsCertExpired))
		return nil
	}
}

// producerAndroidEnterprise handles the resolution side of the Android
// Enterprise "deleted" notification. The emission side happens event-driven
// at the point where the backend detects a 404 from Google's API (see
// server/mdm/android/service/service.go — EmitAndroidEnterpriseDeletedNotification).
//
// This producer resolves the notification whenever Android MDM is
// (re-)configured, so a stale notification from a previous deletion doesn't
// linger after re-setup.
func (svc *Service) producerAndroidEnterprise(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config for android enterprise producer")
	}
	if appCfg.MDM.AndroidEnabledAndConfigured {
		_ = svc.ds.ResolveNotification(ctx, string(fleet.NotificationTypeAndroidEnterpriseDeleted))
	}
	// If not configured we don't emit — that's the job of the event-driven
	// hook in the Android MDM service.
	return nil
}

