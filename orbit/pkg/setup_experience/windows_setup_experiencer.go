package setupexperience

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
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
	// interactiveSession reports the active desktop session; replaced in tests.
	interactiveSession func() (username string, sessionID uint32, present bool, err error)

	// mu guards the fields below; account creation completes on the password
	// page's goroutine, outside Run.
	mu sync.Mutex

	// desktopHidden is set once Explorer is stopped on the provisioning desktop.
	desktopHidden bool
	// cachedIdP holds the IdP identity once the server delivers it, so account
	// creation still works if later config polls stop sending it.
	cachedIdP *fleet.OrbitEndUserIdPAccount

	enabled bool
	// enabledAt is when the setup experience reported enabled.
	enabledAt time.Time
	// reopenBrowser is set when a setup started before this orbit process, so
	// the setup page is opened again (e.g. after a reboot).
	reopenBrowser bool
	// watchingSoftware is set once watchSoftware is polling setup status.
	watchingSoftware   bool
	statusPollInterval time.Duration
	// passwordPage collects the end user's password once software is done.
	passwordPage *passwordPage
	// createdUser is the account created from the IdP identity.
	createdUser string
	finished    bool
	// tearingDown is set while a retried provisioning account teardown runs.
	tearingDown bool

	// pinPendingUser is the created account still awaiting its first login, when
	// fleetd opens Windows Hello PIN setup.
	pinPendingUser string
	// pinSessionSeenAt is when pinPendingUser's session first appeared.
	pinSessionSeenAt time.Time
}

// NewWindowsSetupExperiencer creates the Windows setup experience config receiver.
func NewWindowsSetupExperiencer(client WindowsSetupExperienceInitiator, rootDir string, openBrowser func() error, openURL func(string) error) *WindowsSetupExperiencer {
	s := &WindowsSetupExperiencer{
		OrbitClient: client,
		openBrowser: openBrowser,
		openURL:     openURL,
		rootDir:     rootDir,

		interactiveSession: execuser.InteractiveSession,
		statusPollInterval: softwareStatusPollInterval,
	}
	// The setup experience runs once per host; don't rerun it on later orbit starts.
	info, err := ReadSetupExperienceStatusFile(rootDir)
	if err != nil {
		log.Error().Err(err).Msg("windows setup experience: reading status file")
	}
	switch {
	case info != nil && info.TimeFinished != nil:
		s.finished = true
	case info != nil && info.Enabled:
		// Initiating again would reset the server-side progress of an in-progress setup.
		s.enabled = true
		s.enabledAt = info.TimeInitiated
		s.reopenBrowser = true
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
	if localAccountExists(ProvisioningAccountName) {
		return true
	}
	_, err := os.Stat(filepath.Join(rootDir, pinEnrollmentPendingFilename))
	return err == nil
}

// NeedsFrequentPolling implements fleetclient.FrequentPoller: Run is the clock
// for every step until the PIN prompt opens.
func (s *WindowsSetupExperiencer) NeedsFrequentPolling() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.finished || s.pinPendingUser != ""
}

// Run implements fleet.OrbitConfigReceiver.
func (s *WindowsSetupExperiencer) Run(cfg *fleet.OrbitConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pinPendingUser != "" {
		// The account was created; the provisioning account must be gone before
		// the end user's first login.
		if localAccountExists(ProvisioningAccountName) {
			s.retryTeardown()
			return nil
		}
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
	user, sessionID, present, err := s.interactiveSession()
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
		if err := HideDesktop(sessionID); err != nil {
			log.Error().Err(err).Msg("windows setup experience: hiding provisioning desktop")
		} else {
			s.desktopHidden = true
		}
	}

	// Kick off the server-side setup experience once a session exists. Retry on
	// each poll until it reports enabled: on a first enrollment the setup
	// experience only becomes enabled after Windows MDM enrollment completes,
	// which happens shortly after orbit enrolls. There's no timeout: giving up
	// would hand the end user the provisioning administrator's desktop, so the
	// device waits until the fleet's setup experience is configured.
	if !s.enabled {
		res, err := s.OrbitClient.InitiateSetupExperience()
		if errors.Is(err, fleetclient.ErrMissingLicense) {
			log.Debug().Msg("windows setup experience: premium feature, waiting for a license")
			return nil
		}
		if err != nil {
			log.Error().Err(err).Msg("windows setup experience: initiating")
			return nil
		}
		if !res.Enabled {
			log.Debug().Msg("windows setup experience: not enabled yet, waiting (MDM enrollment may still be completing)")
			return nil
		}
		s.enabled = true
		s.enabledAt = time.Now()
		s.reopenBrowser = true
		if err := WriteSetupExperienceStatusFile(s.rootDir, &SetupExperienceInfo{TimeInitiated: s.enabledAt, Enabled: true}); err != nil {
			log.Error().Err(err).Msg("windows setup experience: writing status file")
		}
		log.Info().Str("user", user).Msg("windows setup experience: initiated")
	}
	if s.reopenBrowser && s.passwordPage == nil {
		s.reopenBrowser = false
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
		// Only open the password page on the provisioning desktop; if that session
		// is gone, Run picks the flow up again once it's back.
		if user, _, present, err := s.interactiveSession(); err == nil && present && strings.EqualFold(user, ProvisioningAccountName) {
			if !s.finished {
				s.startPasswordPage()
			}
		} else {
			s.watchingSoftware = false
		}
		s.mu.Unlock()
		return
	}
}

// setupDone reports whether every setup experience software install and script
// reached a final state, and none failed when all software is required. s.mu
// must be held.
func (s *WindowsSetupExperiencer) setupDone(payload *fleet.SetupExperienceStatusPayload) bool {
	steps := append([]*fleet.SetupExperienceStatusResult{}, payload.Software...)
	if payload.Script != nil {
		steps = append(steps, payload.Script)
	}
	if len(steps) == 0 {
		return time.Since(s.enabledAt) >= emptyStatusGrace
	}
	for _, step := range steps {
		if step == nil {
			continue
		}
		switch step.Status {
		case fleet.SetupExperienceStatusPending, fleet.SetupExperienceStatusRunning:
			return false
		case fleet.SetupExperienceStatusFailure, fleet.SetupExperienceStatusCancelled:
			// The setup page shows the failure and keeps the end user there.
			if payload.RequireAllSoftware {
				return false
			}
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
	username := uniqueUsername(windowsUsernameFromIdP(s.cachedIdP), localAccountExists)
	if username == "" {
		log.Error().Str("email", s.cachedIdP.Email).Msg("windows setup experience: could not derive an available Windows username from IdP identity")
		return
	}
	// The sign-in screen shows the full name, so don't fall back to the email.
	fullname := s.cachedIdP.Fullname
	if fullname == "" {
		fullname = username
	}
	page, err := newPasswordPage(minPasswordLength(), func(password string) error {
		if err := createLocalAccount(username, fullname, password); err != nil {
			return err
		}
		// Record the account right away so a restart resumes at teardown instead
		// of offering the password page again.
		if err := os.WriteFile(s.pinPendingPath(), []byte(username), constant.DefaultFileMode); err != nil {
			log.Error().Err(err).Msg("windows setup experience: recording created account")
		}
		return nil
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
	// If teardown fails, Run retries it on later polls before the PIN prompt.
	if err := teardownProvisioningAccount(ProvisioningAccountName, s.createdUser); err != nil {
		log.Error().Err(err).Msg("windows setup experience: tearing down provisioning account")
	}
	s.pinPendingUser = s.createdUser
	s.finish()
	log.Info().Msg("windows setup experience: complete")
}

// retryTeardown removes a provisioning account left behind by a failed or
// interrupted teardown, outside Run since it waits on logoff. s.mu must be held.
func (s *WindowsSetupExperiencer) retryTeardown() {
	if s.tearingDown {
		return
	}
	s.tearingDown = true
	user := s.pinPendingUser
	go func() {
		err := teardownProvisioningAccount(ProvisioningAccountName, user)
		if err != nil {
			log.Error().Err(err).Msg("windows setup experience: retrying provisioning account teardown")
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.tearingDown = false
		if err == nil && !s.finished {
			s.finish()
		}
	}()
}

// offerPINEnrollment opens Windows Hello PIN setup once on the created
// account's first login.
func (s *WindowsSetupExperiencer) offerPINEnrollment() {
	user, _, present, err := s.interactiveSession()
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
	info := &SetupExperienceInfo{TimeInitiated: time.Now(), Enabled: s.enabled, TimeFinished: new(time.Now())}
	if err := WriteSetupExperienceStatusFile(s.rootDir, info); err != nil {
		log.Error().Err(err).Msg("windows setup experience: writing status file")
	}
}

// uniqueUsername returns name, or name with a numeric suffix if an account by
// that name already exists (e.g. "administrator"), within Windows' 20-character
// limit. It returns "" if no candidate is available.
func uniqueUsername(name string, exists func(string) bool) string {
	if name == "" || !exists(name) {
		return name
	}
	for i := 2; i <= 99; i++ {
		suffix := strconv.Itoa(i)
		base := []rune(name)
		for utf16Len(base)+len(suffix) > 20 {
			base = base[:len(base)-1]
		}
		if candidate := string(base) + suffix; !exists(candidate) {
			return candidate
		}
	}
	return ""
}

func utf16Len(rs []rune) int {
	n := 0
	for _, r := range rs {
		n += utf16.RuneLen(r)
	}
	return n
}

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
