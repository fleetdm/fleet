package openframe

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

type OpenframeTokenRefresher struct {
	tokenExtractor       *OpenframeTokenExtractor
	authorizationManager *OpenFrameAuthorizationManager
	cron                 *cron.Cron
}

func NewOpenframeTokenRefresher(
	tokenExtractor *OpenframeTokenExtractor,
	authorizationManager *OpenFrameAuthorizationManager,
) *OpenframeTokenRefresher {
	return &OpenframeTokenRefresher{
		tokenExtractor:       tokenExtractor,
		authorizationManager: authorizationManager,
		cron:                 cron.New(cron.WithSeconds()),
	}
}

func (tr *OpenframeTokenRefresher) Start() error {
	log.Info().Msg("Scheduling token refresh job")
	_, err := tr.cron.AddFunc("*/5 * * * * *", tr.refreshToken)
	if err != nil {
		return fmt.Errorf("failed to schedule token refresh job: %v", err)
	}
	tr.cron.Start()
	log.Info().Msg("Token refresh job started")
	return nil
}

func (tr *OpenframeTokenRefresher) Stop() {
	if tr.cron != nil {
		log.Info().Msg("Stopping token refresh job")
		ctx := tr.cron.Stop()
		// Wait for running jobs to complete
		<-ctx.Done()
		log.Info().Msg("Token refresh job stopped")
	}
}

func (tr *OpenframeTokenRefresher) refreshToken() {
	log.Info().Msg("Refreshing token")

	token, err := tr.tokenExtractor.ExtractToken()
	if err != nil {
		log.Error().Err(err).Msg("Error extracting token")
		return
	}

	if tr.authorizationManager.GetToken() == token {
		log.Debug().Msg("Openframe token is the same, skipping refresh")
		return
	}

	tr.authorizationManager.UpdateToken(token)
	log.Debug().Msg("Openframe token updated")
}