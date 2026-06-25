package main

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
)

// shouldForceSMTPBackend reports whether a configured (non-SMTP) email backend
// must be cleared because SMTP is already enabled in the app config. SMTP and a
// custom email backend are mutually exclusive, and an already-enabled SMTP
// configuration takes precedence.
func shouldForceSMTPBackend(appCfg *fleet.AppConfig, emailBackend string) bool {
	return appCfg != nil &&
		appCfg.SMTPSettings != nil &&
		appCfg.SMTPSettings.SMTPEnabled &&
		emailBackend != ""
}

// initMailService configures the mail service. Mail is best-effort at startup:
// a construction failure is logged and the (possibly nil) service is returned
// rather than aborting boot.
func initMailService(ctx context.Context, cfg config.FleetConfig, appCfg *fleet.AppConfig, logger *slog.Logger) fleet.MailService {
	if shouldForceSMTPBackend(appCfg, cfg.Email.EmailBackend) {
		// Force-load the SMTP implementation by clearing the configured backend.
		cfg.Email.EmailBackend = ""
		logger.WarnContext(ctx, "SMTP is already enabled, first disable SMTP to utilize a different email backend")
	}

	mailService, err := mail.NewService(cfg)
	if err != nil {
		logger.ErrorContext(ctx, "failed to configure mailing service", "err", err)
	}
	return mailService
}
