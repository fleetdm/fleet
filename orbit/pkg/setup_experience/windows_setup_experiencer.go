package setupexperience

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	fleetclient "github.com/fleetdm/fleet/v4/client"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// ProvisioningAccountName is the temporary local administrator the Windows
// zero-touch image auto-logs-in during OOBE (see the provisioning unattend.xml).
// fleetd runs the setup experience on this account's desktop and deletes it
// during teardown, so the end user's first login is to their own account.
const ProvisioningAccountName = "fleet-provision"

// pinEnrollmentPendingFilename holds the username of the account created from
// the IdP identity until its first login, when fleetd offers Windows Hello PIN
// setup. It's a file so the prompt survives a reboot before that login.
const pinEnrollmentPendingFilename = "windows_pin_enrollment_pending"

// pinEnrollmentDelay gives a brand-new profile time to finish its first-logon
// setup before the PIN prompt opens, so the prompt lands on a ready desktop.
const pinEnrollmentDelay = 30 * time.Second

// softwareStatusPollInterval is how often setup status is checked while
// software installs, faster than the config poll so the password page follows
// the last install within seconds.
const softwareStatusPollInterval = 5 * time.Second

// emptyStatusGrace is how long an empty setup status (no software or scripts)
// is treated as still loading rather than "nothing to install", since the
// server may not have queued the setup steps yet right after initiation.
const emptyStatusGrace = 30 * time.Second

// passwordPageConfirmDelay keeps the confirmation page up briefly before
// the provisioning session is logged off.
const passwordPageConfirmDelay = 5 * time.Second

// WindowsSetupExperienceInitiator kicks off the server-side setup experience and
// returns whether it is enabled for this host.
type WindowsSetupExperienceInitiator interface {
	InitiateSetupExperience() (fleet.SetupExperienceInitResult, error)
	GetSetupExperienceStatus(resetFailedSetupSteps bool) (*fleet.SetupExperienceStatusPayload, error)
}

// WindowsSetupExperiencer drives the Windows zero-touch setup experience from a
// pre-login provisioning desktop. Unlike the one-shot startup path, it is a
// config receiver that retries on every orbit config poll, so it can wait for an
// interactive session to appear (the auto-logon of the provisioning account)
// before launching the IdP sign-in page. After the configured software and
// scripts finish, it asks the end user to choose a password, creates the real
// local account from the IdP identity, and tears down the provisioning account.
type WindowsSetupExperiencer struct {
	OrbitClient WindowsSetupExperienceInitiator
	// openBrowser opens the setup experience (My Device) page in the logged-in
	// user's session.
	openBrowser func() error
	// openURL opens a URL in the setup experience kiosk.
	openURL func(string) error
	rootDir string

	// mu guards the fields below; account creation completes on the password
	// page's goroutine, outside Run.
	mu sync.Mutex

	// provisioningUser is the interactive account the setup experience launched
	// on, captured so teardown deletes the right account.
	provisioningUser string
	// desktopHidden is set once Explorer is stopped on the provisioning desktop.
	desktopHidden bool
	// cachedIdP holds the IdP identity once the server delivers it, so account
	// creation still works if later config polls stop sending it.
	cachedIdP *fleet.OrbitEndUserIdPAccount

	// firstSessionAt is when an interactive session first appeared, used to bound
	// how long we wait for the setup experience to become enabled.
	firstSessionAt time.Time
	enabled        bool
	// enabledAt is when the setup experience reported enabled.
	enabledAt time.Time
	// watchingSoftware is set once watchSoftware is polling setup status.
	watchingSoftware   bool
	statusPollInterval time.Duration
	// passwordPage collects the end user's password once software is done.
	passwordPage *passwordPage
	// createdUser is the account created from the IdP identity.
	createdUser string
	finished    bool

	// pinPendingUser is the created account still awaiting its first login, when
	// fleetd opens Windows Hello PIN setup.
	pinPendingUser string
	// pinSessionSeenAt is when pinPendingUser's session first appeared.
	pinSessionSeenAt time.Time
}

// setupExperienceEnableTimeout bounds how long the receiver retries
// InitiateSetupExperience waiting for it to report enabled. On a first
// enrollment the setup experience only becomes enabled once Windows MDM
// enrollment completes (shortly after orbit enrolls); if it never does, the
// host simply has no setup experience configured.
const setupExperienceEnableTimeout = 10 * time.Minute

// NewWindowsSetupExperiencer creates the Windows setup experience config receiver.
func NewWindowsSetupExperiencer(client WindowsSetupExperienceInitiator, rootDir string, openBrowser func() error, openURL func(string) error) *WindowsSetupExperiencer {
	s := &WindowsSetupExperiencer{
		OrbitClient: client,
		openBrowser: openBrowser,
		openURL:     openURL,
		rootDir:     rootDir,

		statusPollInterval: softwareStatusPollInterval,
	}
	// The setup experience runs once per host; don't rerun it on later orbit starts.
	info, err := ReadSetupExperienceStatusFile(rootDir)
	if err != nil {
		log.Error().Err(err).Msg("windows setup experience: reading status file")
	}
	if info != nil && info.TimeFinished != nil {
		s.finished = true
	}
	if b, err := os.ReadFile(s.pinPendingPath()); err == nil {
		s.pinPendingUser = strings.TrimSpace(string(b))
	}
	return s
}

func (s *WindowsSetupExperiencer) pinPendingPath() string {
	return filepath.Join(s.rootDir, pinEnrollmentPendingFilename)
}

// WindowsZeroTouchActive reports whether this Windows host uses the zero-touch
// flow: its image created the provisioning account, or the account created
// from the IdP identity is still waiting for its PIN prompt. Other hosts keep
// the one-shot setup experience.
func WindowsZeroTouchActive(rootDir string) bool {
	if provisioningAccountExists() {
		return true
	}
	_, err := os.Stat(filepath.Join(rootDir, pinEnrollmentPendingFilename))
	return err == nil
}

// Run implements fleet.OrbitConfigReceiver.
func (s *WindowsSetupExperiencer) Run(cfg *fleet.OrbitConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pinPendingUser != "" {
		s.offerPINEnrollment()
		return nil
	}
	if s.finished {
		return nil
	}

	// Remember the IdP identity whenever the server sends it; it's needed after
	// the setup experience completes, by which point the server may stop sending it.
	if cfg.EndUserIdPAccount != nil {
		s.cachedIdP = cfg.EndUserIdPAccount
	}

	// Unlike macOS/Autopilot, the Windows setup experience is not driven by the
	// RunSetupExperience notification (that is the awaiting-configuration/ESP
	// path). fleetd initiates it itself; InitiateSetupExperience below reports
	// whether it is enabled for this host's fleet.

	// Wait for the provisioning account to be logged in before doing anything
	// that needs a desktop. During OOBE the active session belongs to
	// defaultuser0, which interactiveSession filters out.
	user, sessionID, present, err := execuser.InteractiveSession()
	if err != nil {
		log.Error().Err(err).Msg("windows setup experience: checking interactive session")
		return nil
	}
	if !present {
		log.Debug().Msg("windows setup experience: waiting for interactive session")
		return nil
	}
	// Only drive the flow on the provisioning desktop; never open the kiosk or
	// create an account in someone else's session.
	if !strings.EqualFold(user, ProvisioningAccountName) {
		log.Debug().Str("user", user).Msg("windows setup experience: waiting for the provisioning account session")
		return nil
	}

	// The provisioning desktop is only a backdrop for the kiosk, so hide its icons
	// and taskbar.
	if !s.desktopHidden {
		if err := hideDesktop(sessionID); err != nil {
			log.Error().Err(err).Msg("windows setup experience: hiding provisioning desktop")
		} else {
			s.desktopHidden = true
		}
	}

	// Kick off the server-side setup experience once a session exists. Retry on
	// each poll until it reports enabled: on a first enrollment the setup
	// experience only becomes enabled after Windows MDM enrollment completes,
	// which happens shortly after orbit enrolls. Bounded by a timeout so a host
	// with no setup experience configured doesn't loop forever.
	if !s.enabled {
		if s.firstSessionAt.IsZero() {
			s.firstSessionAt = time.Now()
			s.provisioningUser = user
		}
		res, err := s.OrbitClient.InitiateSetupExperience()
		if errors.Is(err, fleetclient.ErrMissingLicense) {
			// Setup experience is a premium feature; nothing to do on this host.
			log.Debug().Msg("windows setup experience: premium feature, skipping")
			s.finish()
			return nil
		}
		if err != nil {
			log.Error().Err(err).Msg("windows setup experience: initiating")
			return nil
		}
		if !res.Enabled {
			if time.Since(s.firstSessionAt) > setupExperienceEnableTimeout {
				log.Info().Msg("windows setup experience: not enabled within timeout, nothing to do")
				s.finish()
				return nil
			}
			log.Debug().Msg("windows setup experience: not enabled yet, waiting (MDM enrollment may still be completing)")
			return nil
		}
		s.enabled = true
		s.enabledAt = time.Now()
		log.Info().Str("user", user).Msg("windows setup experience: initiated")
		if s.openBrowser != nil {
			if err := s.openBrowser(); err != nil {
				log.Error().Err(err).Msg("windows setup experience: opening setup page")
			}
		}
	}

	// Poll until the configured software and scripts reach a terminal state.
	payload, err := s.OrbitClient.GetSetupExperienceStatus(false)
	if err != nil {
		log.Error().Err(err).Msg("windows setup experience: getting status")
		return nil
	}
	if !s.setupDone(payload) {
		log.Debug().Msg("windows setup experience: software/scripts still running")
		if !s.watchingSoftware {
			s.watchingSoftware = true
			go s.watchSoftware()
		}
		return nil
	}
	s.startPasswordPage()
	return nil
}

// watchSoftware polls setup status until software and scripts are done, then
// starts the password page.
func (s *WindowsSetupExperiencer) watchSoftware() {
	ticker := time.NewTicker(s.statusPollInterval)
	defer ticker.Stop()
	for range ticker.C {
		payload, err := s.OrbitClient.GetSetupExperienceStatus(false)
		if err != nil {
			log.Debug().Err(err).Msg("windows setup experience: getting status")
			continue
		}
		s.mu.Lock()
		done := s.setupDone(payload)
		s.mu.Unlock()
		if !done {
			continue
		}
		s.mu.Lock()
		if !s.finished {
			s.startPasswordPage()
		}
		s.mu.Unlock()
		return
	}
}

// setupDone reports whether every setup experience software install and script
// reached a final state. s.mu must be held.
func (s *WindowsSetupExperiencer) setupDone(payload *fleet.SetupExperienceStatusPayload) bool {
	steps := append([]*fleet.SetupExperienceStatusResult{}, payload.Software...)
	if payload.Script != nil {
		steps = append(steps, payload.Script)
	}
	if len(steps) == 0 {
		return time.Since(s.enabledAt) >= emptyStatusGrace
	}
	for _, step := range steps {
		if step != nil && (step.Status == fleet.SetupExperienceStatusPending || step.Status == fleet.SetupExperienceStatusRunning) {
			return false
		}
	}
	return true
}

// startPasswordPage asks the end user to choose a password once software and
// scripts are done; the page creates the account and completeSetup tears down
// the provisioning account. s.mu must be held.
func (s *WindowsSetupExperiencer) startPasswordPage() {
	if s.passwordPage != nil {
		return
	}
	if s.cachedIdP == nil {
		log.Info().Msg("windows setup experience: software done, waiting for IdP identity")
		return
	}
	username := windowsUsernameFromIdP(s.cachedIdP)
	if username == "" {
		log.Error().Str("email", s.cachedIdP.Email).Msg("windows setup experience: could not derive a Windows username from IdP identity")
		return
	}
	// The sign-in screen shows the full name, so don't fall back to the email.
	fullname := s.cachedIdP.Fullname
	if fullname == "" {
		fullname = username
	}
	page, err := newPasswordPage(minPasswordLength(), func(password string) error {
		return createLocalAccount(username, fullname, password)
	})
	if err != nil {
		log.Error().Err(err).Msg("windows setup experience: starting password page")
		return
	}
	if err := s.openURL(page.URL()); err != nil {
		log.Error().Err(err).Msg("windows setup experience: opening password page")
		_ = page.Close()
		return
	}
	s.passwordPage = page
	log.Info().Str("username", username).Msg("windows setup experience: waiting for end user to choose a password")
	go s.completeSetup(page, username)
}

// completeSetup waits for the password page to create the account, then tears
// down the provisioning account.
func (s *WindowsSetupExperiencer) completeSetup(page *passwordPage, username string) {
	<-page.Done()
	// Let the kiosk show the confirmation before the session is logged off.
	time.Sleep(passwordPageConfirmDelay)
	_ = page.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.createdUser = username
	if s.provisioningUser == ProvisioningAccountName {
		// The account exists now, so finish even if teardown fails: the end user
		// can still sign in with it.
		if err := teardownProvisioningAccount(s.provisioningUser, s.createdUser); err != nil {
			log.Error().Err(err).Msg("windows setup experience: tearing down provisioning account")
		}
	} else {
		log.Info().Str("user", s.provisioningUser).Msg("windows setup experience: not deleting non-provisioning account")
	}

	if err := os.WriteFile(s.pinPendingPath(), []byte(s.createdUser), constant.DefaultFileMode); err != nil {
		log.Error().Err(err).Msg("windows setup experience: recording pending PIN enrollment")
	} else {
		s.pinPendingUser = s.createdUser
	}

	s.finish()
	log.Info().Msg("windows setup experience: complete")
}

// offerPINEnrollment opens Windows Hello PIN setup once on the created
// account's first login.
func (s *WindowsSetupExperiencer) offerPINEnrollment() {
	user, _, present, err := execuser.InteractiveSession()
	if err != nil || !present || !strings.EqualFold(user, s.pinPendingUser) {
		s.pinSessionSeenAt = time.Time{}
		return
	}
	if s.pinSessionSeenAt.IsZero() {
		s.pinSessionSeenAt = time.Now()
	}
	if time.Since(s.pinSessionSeenAt) < pinEnrollmentDelay {
		return
	}
	if err := openPINEnrollment(); err != nil {
		log.Error().Err(err).Msg("windows setup experience: opening PIN enrollment")
		return
	}
	if err := os.Remove(s.pinPendingPath()); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("windows setup experience: clearing pending PIN enrollment")
	}
	log.Info().Str("user", s.pinPendingUser).Msg("windows setup experience: opened PIN enrollment")
	s.pinPendingUser = ""
}

func (s *WindowsSetupExperiencer) finish() {
	s.finished = true
	// Give the desktop back if the setup experience ended without tearing down
	// the provisioning session (for example, it was never enabled).
	if s.desktopHidden && s.createdUser == "" {
		if err := showDesktop(); err != nil {
			log.Error().Err(err).Msg("windows setup experience: restoring provisioning desktop")
		}
	}
	info := &SetupExperienceInfo{TimeInitiated: time.Now(), Enabled: s.enabled, TimeFinished: ptrTime(time.Now())}
	if err := WriteSetupExperienceStatusFile(s.rootDir, info); err != nil {
		log.Error().Err(err).Msg("windows setup experience: writing status file")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// windowsUsernameFromIdP derives a valid local Windows username (<=20 chars, no
// reserved characters) from the IdP identity, preferring the IdP username and
// falling back to the local part of the email.
func windowsUsernameFromIdP(idp *fleet.OrbitEndUserIdPAccount) string {
	base := idp.Username
	if base == "" {
		base = idp.Email
	}
	if i := strings.IndexByte(base, '@'); i >= 0 {
		base = base[:i]
	}
	// Drop any plus-address tag (e.g. "luke+windows" -> "luke").
	if i := strings.IndexByte(base, '+'); i >= 0 {
		base = base[:i]
	}
	// Windows usernames can't contain " / \ [ ] : ; | = , + * ? < > or spaces.
	var b strings.Builder
	for _, r := range base {
		if strings.ContainsRune(`"/\[]:;|=,+*?<> `, r) {
			continue
		}
		b.WriteRune(r)
	}
	// Windows allows 20 UTF-16 code units; cut on a rune boundary.
	var name []rune
	units := 0
	for _, r := range b.String() {
		if units+utf16.RuneLen(r) > 20 {
			break
		}
		units += utf16.RuneLen(r)
		name = append(name, r)
	}
	return string(name)
}
