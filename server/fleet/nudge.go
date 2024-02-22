package fleet

import (
	"time"
)

// NudgeConfig represents a subset of the [full Nudge configuration][0] that we
// want to override.
//
// [0]: https://github.com/macadmins/nudge/wiki
type NudgeConfig struct {
	OSVersionRequirements []nudgeOSVersionRequirements `json:"osVersionRequirements"`
	UserInterface         nudgeUserInterface           `json:"userInterface"`
	UserExperience        nudgeUserExperience          `json:"userExperience"`
}

type nudgeAboutUpdateURLs struct {
	Language       string `json:"_language"`
	AboutUpdateURL string `json:"aboutUpdateURL"`
}

type nudgeOSVersionRequirements struct {
	RequiredInstallationDate time.Time              `json:"requiredInstallationDate"`
	RequiredMinimumOSVersion string                 `json:"requiredMinimumOSVersion"`
	AboutUpdateURLs          []nudgeAboutUpdateURLs `json:"aboutUpdateURLs"`
}

type nudgeUserInterface struct {
	SimpleMode        bool                  `json:"simpleMode"`
	ShowDeferralCount bool                  `json:"showDeferralCount"`
	UpdateElements    []nudgeUpdateElements `json:"updateElements"`
}

type nudgeUserExperience struct {
	InitialRefreshCycle     int `json:"initialRefreshCycle"`
	ApproachingRefreshCycle int `json:"approachingRefreshCycle"`
	ImminentRefreshCycle    int `json:"imminentRefreshCycle"`
	ElapsedRefreshCycle     int `json:"elapsedRefreshCycle"`
}

type nudgeUpdateElements struct {
	Language         string `json:"_language"`
	ActionButtonText string `json:"actionButtonText"`
	MainHeader       string `json:"mainHeader"`
}

func NewNudgeConfig(macOSUpdates MacOSUpdates) (*NudgeConfig, error) {
	deadline, err := time.Parse("2006-01-02", macOSUpdates.Deadline.Value)
	if err != nil {
		return nil, err
	}

	// Per the spec, the exact deadline time is arbitrarily chosen to be
	// 04:00:00 (UTC-8) until we allow users to customize it.
	//
	// See https://github.com/fleetdm/fleet/issues/9013 for more details.
	localizedDeadline := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 4, 0, 0, 0, time.UTC)

	return &NudgeConfig{
		OSVersionRequirements: []nudgeOSVersionRequirements{{
			RequiredInstallationDate: localizedDeadline,
			RequiredMinimumOSVersion: macOSUpdates.MinimumVersion.Value,
			AboutUpdateURLs: []nudgeAboutUpdateURLs{{
				Language:       "en",
				AboutUpdateURL: "https://fleetdm.com/learn-more-about/os-updates",
			}},
		}},
		UserInterface: nudgeUserInterface{
			SimpleMode:        true,
			ShowDeferralCount: false,
			UpdateElements: []nudgeUpdateElements{{
				Language:         "en",
				ActionButtonText: "Update",
				MainHeader:       "Your device requires an update",
			}},
		},
		UserExperience: nudgeUserExperience{
			/* Initially, we show Nudge once every 24 hours  */
			InitialRefreshCycle: 86400,
			/*
			 * Related to approachingWindowTime (72 hours before deadline by default)
			 * we still want to show the window once every 24 hours.
			 */
			ApproachingRefreshCycle: 86400,
			/*
			 * Related to imminentWindowTime (24 hours before deadline by default)
			 * we want to show the window once every 2 hours.
			 */
			ImminentRefreshCycle: 7200,
			/*
			 * Related to elapsedWindowTime (once the deadline is past)
			 * we want to show the window once every hour.
			 */
			ElapsedRefreshCycle: 3600,
		},
	}, nil
}
