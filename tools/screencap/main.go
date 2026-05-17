package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// modal describes a button to click to open a modal on a page,
// with optional tabs to click through inside the modal.
type modal struct {
	name     string
	selector string
	tabs     []modalTab
}

type modalTab struct {
	name     string
	selector string
}

type pageEntry struct {
	path   string
	modals []modal
}

var addHostsModal = modal{
	"add-hosts", `button[class*="add-hosts"]`,
	[]modalTab{
		{"macos", `[data-text="macOS"]`},
		{"windows", `[data-text="Windows"]`},
		{"linux", `[data-text="Linux"]`},
		{"chromeos", `[data-text="ChromeOS"]`},
		{"ios-ipados", `[data-text="iOS & iPadOS"]`},
		{"android", `[data-text="Android"]`},
		{"advanced", `[data-text="Advanced"]`},
	},
}

// fullPages is the built-in "full" workflow.
var fullPages = []pageEntry{
	{"/dashboard", []modal{addHostsModal}},
	{"/dashboard/linux", nil},
	{"/dashboard/mac", nil},
	{"/dashboard/windows", nil},
	{"/dashboard/chrome", nil},
	{"/dashboard/ios", nil},
	{"/dashboard/ipados", nil},
	{"/hosts/manage", []modal{
		addHostsModal,
		{"edit-columns", `button[class*="edit-columns-button"]`, nil},
	}},
	{"/labels/manage", nil},
	{"/queries/manage", []modal{
		{"manage-automations", `button[class*="manage-automations"]`, nil},
	}},
	{"/queries/new", nil},
	{"/policies/manage", []modal{
		{"manage-automations", `button[class*="manage-automations"]`, nil},
	}},
	{"/policies/new", nil},
	{"/software/titles", nil},
	{"/software/os", nil},
	{"/software/versions", nil},
	{"/software/vulnerabilities", nil},
	{"/controls", nil},
	{"/controls/os-updates", nil},
	{"/controls/os-settings", nil},
	{"/controls/os-settings/custom-settings", nil},
	{"/controls/os-settings/certificates", nil},
	{"/controls/os-settings/disk-encryption", nil},
	{"/controls/setup-experience", nil},
	{"/controls/scripts", nil},
	{"/controls/scripts/library", nil},
	{"/controls/variables", []modal{
		{"add-variable", `button[class*="add-secret"]`, nil},
	}},
	{"/settings/organization/info", nil},
	{"/settings/organization/webaddress", nil},
	{"/settings/organization/smtp", nil},
	{"/settings/organization/agents", nil},
	{"/settings/organization/statistics", nil},
	{"/settings/organization/advanced", nil},
	{"/settings/organization/fleet-desktop", nil},
	{"/settings/users", []modal{
		{"add-user", `.action-button__add-user`, nil},
	}},
	{"/settings/teams", []modal{
		{"create-team", `.action-button__create-team`, nil},
	}},
	{"/settings/integrations/ticket-destinations", []modal{
		{"add-integration", `.action-button__add-integration`, nil},
	}},
	{"/settings/integrations/mdm", nil},
	{"/settings/integrations/calendars", nil},
	{"/settings/integrations/change-management", nil},
	{"/settings/integrations/conditional-access", nil},
	{"/settings/integrations/certificate-authorities", nil},
	{"/settings/integrations/identity-provider", nil},
	{"/settings/integrations/sso", nil},
	{"/settings/integrations/host-status-webhook", nil},
	{"/account", []modal{
		{"change-password", `button[class*="change-password"]`, nil},
		{"get-api-token", `button[class*="api-token"]`, nil},
	}},
}

func main() {
	sso := flag.Bool("sso", false, "use SSO login (opens visible browser)")
	cookie := flag.String("cookie", "", "session cookie value")
	cookieName := flag.String("cookie-name", "Fleet-Session", "name of the session cookie")
	email := flag.String("email", "", "login email")
	password := flag.String("password", "", "login password")
	waitSec := flag.Int("wait-time-seconds", 6, "seconds to wait for each page to load")
	loginFlag := flag.Bool("login", false, "force a new login")
	record := flag.String("record", "", "record a new workflow with this name")
	run := flag.String("workflow", "", "run a saved workflow (use 'full' for built-in)")
	list := flag.Bool("list", false, "list saved workflows")
	flag.Parse()

	if *list {
		names, err := listWorkflows()
		if err != nil {
			log.Fatalf("Error listing workflows: %v", err)
		}
		fmt.Println("Saved workflows:")
		fmt.Println("  full (built-in)")
		for _, n := range names {
			fmt.Printf("  %s\n", n)
		}
		return
	}

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	useSSO := *sso
	useCookie := *cookie != ""
	usePassword := *email != "" && *password != ""
	needsLogin := useSSO || useCookie || usePassword || *loginFlag
	isRecording := *record != ""

	waitTime := time.Duration(*waitSec) * time.Second

	baseURL := strings.TrimRight(flag.Arg(0), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("Invalid base URL %q: %v", baseURL, err)
	}

	// Use a persistent Chrome profile so sessions survive across runs.
	profileDir, err := chromeProfileDir()
	if err != nil {
		log.Fatalf("Error creating Chrome profile directory: %v", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.WindowSize(1440, 900),
		chromedp.UserDataDir(profileDir),
	)

	// Show browser for SSO, login, or recording.
	if useSSO || *loginFlag || isRecording {
		opts = append(opts, chromedp.Flag("headless", false))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	if needsLogin {
		switch {
		case useSSO || *loginFlag:
			loginSSO(ctx, baseURL, waitTime)
		case useCookie:
			loginWithCookie(ctx, baseURL, parsed, *cookieName, *cookie, waitTime)
		case usePassword:
			loginPassword(ctx, baseURL, *email, *password, waitTime)
		}
	} else {
		fmt.Println("Using saved session. (Use -sso, -email/-password, or -login to re-authenticate.)")
		if err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/dashboard"),
			chromedp.Sleep(waitTime),
		); err != nil {
			log.Fatalf("Failed to load dashboard with saved session: %v\nTry running again with -sso or -email/-password to log in.", err)
		}
	}

	if err := chromedp.Run(ctx, page.SetLifecycleEventsEnabled(true)); err != nil {
		log.Fatalf("Failed to enable lifecycle events: %v", err)
	}

	if isRecording {
		recordWorkflow(ctx, baseURL, parsed, *record, waitTime)
		return
	}

	// Determine which workflow to run.
	workflowName := *run
	if workflowName == "" {
		workflowName = "full"
	}

	// Build output directory.
	host := strings.ReplaceAll(parsed.Host, ":", "-")
	dirName := fmt.Sprintf("%s-%s-%s", time.Now().Format("2006-01-02_150405"), workflowName, host)
	outDir, err := filepath.Abs(filepath.Join("screenshots", dirName))
	if err != nil {
		log.Fatalf("Error resolving output directory: %v", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("Error creating output directory %s: %v", outDir, err)
	}
	fmt.Printf("Saving screenshots to %s\n", outDir)

	if workflowName == "full" {
		runFullWorkflow(ctx, baseURL, outDir, waitTime)
	} else {
		runSavedWorkflow(ctx, baseURL, outDir, workflowName, waitTime)
	}
}

// recordWorkflow opens a visible browser and lets the user click around.
// It injects JS listeners to capture clicks, radio/checkbox toggles, tab
// switches, and tooltip hovers. Each time the user presses ENTER, the
// current page state and all recorded actions since the last step are saved.
func recordWorkflow(ctx context.Context, baseURL string, parsed *url.URL, name string, waitTime time.Duration) {
	// Inject the recorder script into the current page.
	injectRecorder(ctx)

	// Re-inject on every page navigation (the script doesn't survive navigations).
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if _, ok := ev.(*page.EventLoadEventFired); ok {
			injectRecorder(ctx)
		}
	})

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Printf("  Recording workflow: %s\n", name)
	fmt.Println()
	fmt.Println("  Browse around in the browser window.")
	fmt.Println("  Clicks, radio buttons, checkboxes, tabs, and tooltip")
	fmt.Println("  hovers are recorded automatically.")
	fmt.Println()
	fmt.Println("  Press ENTER to save a screenshot step.")
	fmt.Println("  Type a name before ENTER to name the step.")
	fmt.Println("  Type 'done' to finish and save the workflow.")
	fmt.Println("============================================================")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	w := &workflow{Name: name}
	stepNum := 0
	var prevPath string

	for {
		fmt.Print("  [ENTER to capture, 'done' to finish] > ")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "done" {
			break
		}

		// Get current page state from the browser.
		var currentURL string
		var scrollY float64
		var hasModal bool
		var modalTitle string

		if err := chromedp.Run(ctx,
			chromedp.Evaluate(`window.location.href`, &currentURL),
			chromedp.Evaluate(`window.scrollY`, &scrollY),
			chromedp.Evaluate(`document.querySelector('.modal__modal_container') !== null`, &hasModal),
		); err != nil {
			log.Printf("  Error reading page state: %v", err)
			continue
		}

		if hasModal {
			chromedp.Run(ctx,
				chromedp.Evaluate(`
					(() => {
						const h = document.querySelector('.modal__header');
						return h ? h.textContent.trim() : '';
					})()
				`, &modalTitle),
			)
		}

		// Drain recorded actions from JS.
		var rawActions []recordedAction
		chromedp.Run(ctx, chromedp.Evaluate(drainActionsJS, &rawActions))

		// Extract the path relative to the base URL.
		currentParsed, err := url.Parse(currentURL)
		if err != nil {
			log.Printf("  Error parsing URL: %v", err)
			continue
		}
		path := currentParsed.Path
		if currentParsed.RawQuery != "" {
			path += "?" + currentParsed.RawQuery
		}

		// If the page changed, only keep the actions from this page
		// (the ones before navigation are stale).
		var actions []recordedAction
		if path == prevPath || prevPath == "" {
			actions = rawActions
		} else {
			// Page navigated — actions from the old page can't be replayed
			// on this new page, so we drop them.
			actions = rawActions
		}
		prevPath = path

		stepNum++
		stepName := fmt.Sprintf("step-%d", stepNum)
		if input != "" {
			stepName = input
		}

		step := workflowStep{
			Name:       stepName,
			Path:       path,
			ScrollY:    scrollY,
			HasModal:   hasModal,
			ModalTitle: modalTitle,
			Actions:    actions,
		}
		w.Steps = append(w.Steps, step)

		desc := path
		if len(actions) > 0 {
			desc += fmt.Sprintf(" (%d actions", len(actions))
			for _, a := range actions {
				desc += fmt.Sprintf(" [%s: %s]", a.Kind, truncate(a.Text, 30))
			}
			desc += ")"
		}
		if hasModal {
			desc += fmt.Sprintf(" (modal: %s)", modalTitle)
		}
		if scrollY > 0 {
			desc += fmt.Sprintf(" (scrollY: %.0f)", scrollY)
		}
		fmt.Printf("  Step %d saved: %s -> %s\n", stepNum, stepName, desc)
	}

	if len(w.Steps) == 0 {
		fmt.Println("No steps recorded. Workflow not saved.")
		return
	}

	if err := saveWorkflow(w); err != nil {
		log.Fatalf("Error saving workflow: %v", err)
	}
	fmt.Printf("\nWorkflow %q saved with %d steps.\n", name, len(w.Steps))
	fmt.Println("Run it with:")
	fmt.Printf("  make build && ./screencap -workflow %s <base-url>\n", name)
}

// injectRecorder injects the action recorder JS into the current page.
func injectRecorder(ctx context.Context) {
	chromedp.Run(ctx, chromedp.Evaluate(recorderJS, nil))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// runSavedWorkflow replays a previously recorded workflow.
func runSavedWorkflow(ctx context.Context, baseURL, outDir, name string, waitTime time.Duration) {
	w, err := loadWorkflow(name)
	if err != nil {
		log.Fatalf("Error loading workflow: %v", err)
	}

	fmt.Printf("Running workflow %q (%d steps)\n", w.Name, len(w.Steps))

	for i, step := range w.Steps {
		pageURL := baseURL + step.Path
		fmt.Printf("[%d/%d] %s -> %s\n", i+1, len(w.Steps), step.Name, step.Path)

		if err := chromedp.Run(ctx,
			chromedp.Navigate(pageURL),
			waitForNetworkIdle(ctx, waitTime),
		); err != nil {
			log.Printf("  ERROR loading %s: %v", step.Path, err)
			continue
		}

		// Replay recorded actions before taking the screenshot.
		for _, action := range step.Actions {
			replayAction(ctx, action)
		}

		// Restore scroll position if recorded.
		if step.ScrollY > 0 {
			chromedp.Run(ctx,
				chromedp.Evaluate(fmt.Sprintf(`window.scrollTo(0, %f)`, step.ScrollY), nil),
			)
			time.Sleep(250 * time.Millisecond)
		}

		buf, err := takeScreenshot(ctx)
		if err != nil {
			log.Printf("  ERROR capturing %s: %v", step.Name, err)
			continue
		}

		filename := fmt.Sprintf("%s-1.png", step.Name)
		writeScreenshot(outDir, filename, buf)
	}

	fmt.Printf("\nDone! %d screenshots saved to %s\n", len(w.Steps), outDir)
}

// replayAction replays a single recorded action in the browser.
func replayAction(ctx context.Context, action recordedAction) {
	if !elementExists(ctx, action.Selector) {
		log.Printf("    action %s: selector %q not found, skipping", action.Kind, truncate(action.Selector, 50))
		return
	}

	switch action.Kind {
	case actionHover:
		if err := chromedp.Run(ctx,
			chromedp.ScrollIntoView(action.Selector, chromedp.ByQuery),
			chromedp.ActionFunc(func(ctx context.Context) error {
				// Dispatch a mouseover event to trigger tooltips.
				return chromedp.Evaluate(fmt.Sprintf(`(() => {
					const el = document.querySelector(%q);
					if (el) {
						el.dispatchEvent(new MouseEvent('mouseover', {bubbles: true}));
						el.dispatchEvent(new MouseEvent('mouseenter', {bubbles: true}));
					}
				})()`, action.Selector), nil).Do(ctx)
			}),
		); err != nil {
			log.Printf("    action hover: %v", err)
			return
		}
		// Wait for tooltip to appear.
		time.Sleep(500 * time.Millisecond)

	case actionClick, actionRadio, actionCheckbox, actionSelectTab, actionToggle:
		if err := chromedp.Run(ctx,
			chromedp.ScrollIntoView(action.Selector, chromedp.ByQuery),
			chromedp.Click(action.Selector, chromedp.ByQuery),
		); err != nil {
			log.Printf("    action %s: %v", action.Kind, err)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Printf("    replayed %s on %s\n", action.Kind, truncate(action.Selector, 60))
}

// runFullWorkflow runs the built-in full page + modal workflow.
func runFullWorkflow(ctx context.Context, baseURL, outDir string, waitTime time.Duration) {
	for i, pg := range fullPages {
		pageURL := baseURL + pg.path
		baseName := pathToFilename(pg.path)

		fmt.Printf("[%d/%d] %s\n", i+1, len(fullPages), pg.path)

		if err := chromedp.Run(ctx,
			chromedp.Navigate(pageURL),
			waitForNetworkIdle(ctx, waitTime),
		); err != nil {
			log.Printf("  ERROR loading %s: %v", pg.path, err)
			continue
		}

		capturePage(ctx, outDir, baseName)

		for _, m := range pg.modals {
			captureModal(ctx, outDir, baseName, m, pageURL, waitTime)
		}
	}

	fmt.Printf("\nDone! Screenshots saved to %s\n", outDir)
}

// capturePage scrolls through the page and captures one screenshot per viewport.
func capturePage(ctx context.Context, outDir, baseName string) {
	var docHeight, viewportHeight float64
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.documentElement.scrollHeight`, &docHeight),
		chromedp.Evaluate(`window.innerHeight`, &viewportHeight),
	); err != nil {
		log.Printf("  ERROR getting dimensions: %v", err)
		return
	}

	part := 1
	for offset := 0.0; offset < docHeight; offset += viewportHeight {
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`window.scrollTo(0, %f)`, offset), nil),
		); err != nil {
			log.Printf("  ERROR scrolling: %v", err)
			break
		}
		time.Sleep(250 * time.Millisecond)

		buf, err := takeScreenshot(ctx)
		if err != nil {
			log.Printf("  ERROR capturing part %d: %v", part, err)
			break
		}

		filename := fmt.Sprintf("%s-%d.png", baseName, part)
		if err := os.WriteFile(filepath.Join(outDir, filename), buf, 0o644); err != nil {
			log.Printf("  ERROR writing %s: %v", filename, err)
			break
		}
		part++
	}
}

func captureModal(ctx context.Context, outDir, baseName string, m modal, pageURL string, waitTime time.Duration) {
	if !elementExists(ctx, m.selector) {
		log.Printf("  modal %q: button not found (%s), skipping", m.name, m.selector)
		return
	}

	if err := chromedp.Run(ctx,
		chromedp.ScrollIntoView(m.selector, chromedp.ByQuery),
		chromedp.Click(m.selector, chromedp.ByQuery),
	); err != nil {
		log.Printf("  modal %q: failed to click: %v", m.name, err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	if !elementExists(ctx, ".modal__modal_container") {
		log.Printf("  modal %q: no modal appeared after click, skipping", m.name)
		return
	}

	fmt.Printf("    modal: %s\n", m.name)

	if buf, err := takeScreenshot(ctx); err != nil {
		log.Printf("  modal %q: capture failed: %v", m.name, err)
	} else {
		filename := fmt.Sprintf("%s-modal-%s-1.png", baseName, m.name)
		writeScreenshot(outDir, filename, buf)
	}

	for _, tab := range m.tabs {
		if !elementExists(ctx, tab.selector) {
			log.Printf("    tab %q: not found (%s), skipping", tab.name, tab.selector)
			continue
		}

		if err := chromedp.Run(ctx,
			chromedp.Click(tab.selector, chromedp.ByQuery),
		); err != nil {
			log.Printf("    tab %q: failed to click: %v", tab.name, err)
			continue
		}

		time.Sleep(300 * time.Millisecond)
		fmt.Printf("      tab: %s\n", tab.name)

		if buf, err := takeScreenshot(ctx); err != nil {
			log.Printf("    tab %q: capture failed: %v", tab.name, err)
		} else {
			filename := fmt.Sprintf("%s-modal-%s-%s-1.png", baseName, m.name, tab.name)
			writeScreenshot(outDir, filename, buf)
		}
	}

	closeModal(ctx)

	chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		waitForNetworkIdle(ctx, waitTime),
	)
}

func elementExists(ctx context.Context, selector string) bool {
	var exists bool
	chromedp.Run(ctx,
		chromedp.Evaluate(fmt.Sprintf(`document.querySelector(%q) !== null`, selector), &exists),
	)
	return exists
}

func closeModal(ctx context.Context) {
	chromedp.Run(ctx, chromedp.Evaluate(`
		(() => {
			const closeBtn = document.querySelector('.modal__ex');
			if (closeBtn) { closeBtn.click(); return; }
			document.dispatchEvent(new KeyboardEvent('keydown', {key: 'Escape', bubbles: true}));
		})()
	`, nil))
	time.Sleep(300 * time.Millisecond)
}

func writeScreenshot(outDir, filename string, buf []byte) {
	if err := os.WriteFile(filepath.Join(outDir, filename), buf, 0o644); err != nil {
		log.Printf("  ERROR writing %s: %v", filename, err)
	}
}

func takeScreenshot(ctx context.Context) ([]byte, error) {
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				Do(ctx)
			return err
		}),
	)
	return buf, err
}

func chromeProfileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".fleet", "screencap-profile")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func loginSSO(ctx context.Context, baseURL string, waitTime time.Duration) {
	loginURL := baseURL + "/login"
	fmt.Printf("Navigating to %s ...\n", loginURL)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible(`.login-form__sso-btn`, chromedp.ByQuery),
		chromedp.Click(`.login-form__sso-btn`, chromedp.ByQuery),
	); err != nil {
		log.Fatalf("Failed to initiate SSO login: %v", err)
	}

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println("  Complete the SSO login in the browser window that opened.")
	fmt.Println("  Once you see the Fleet dashboard, press ENTER here.")
	fmt.Println("============================================================")
	fmt.Println()

	bufio.NewReader(os.Stdin).ReadBytes('\n')

	if err := chromedp.Run(ctx, chromedp.Sleep(waitTime)); err != nil {
		log.Fatalf("Error after SSO login: %v", err)
	}
	fmt.Println("SSO login complete.")
}

func loginWithCookie(ctx context.Context, baseURL string, parsed *url.URL, cookieName, cookieValue string, waitTime time.Duration) {
	fmt.Printf("Setting cookie %s on %s ...\n", cookieName, parsed.Host)

	secure := parsed.Scheme == "https"

	if err := chromedp.Run(ctx,
		network.SetCookie(cookieName, cookieValue).
			WithDomain(parsed.Hostname()).
			WithPath("/").
			WithHTTPOnly(true).
			WithSecure(secure),
		chromedp.Navigate(baseURL+"/dashboard"),
		chromedp.Sleep(waitTime),
	); err != nil {
		log.Fatalf("Failed to set cookie: %v", err)
	}
	fmt.Println("Cookie set. Session loaded.")
}

func loginPassword(ctx context.Context, baseURL, email, password string, waitTime time.Duration) {
	loginURL := baseURL + "/login"
	fmt.Printf("Logging in at %s ...\n", loginURL)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible(`input[placeholder="Email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[placeholder="Email"]`, email, chromedp.ByQuery),
		chromedp.SendKeys(`input[placeholder="Password"]`, password, chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(waitTime),
	); err != nil {
		log.Fatalf("Failed to log in: %v", err)
	}
	fmt.Println("Login successful.")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: screencap [flags] <base-url>\n\n")
	fmt.Fprintf(os.Stderr, "Run the built-in full workflow:\n")
	fmt.Fprintf(os.Stderr, "  screencap https://fleet.example.com\n\n")
	fmt.Fprintf(os.Stderr, "Record a custom workflow:\n")
	fmt.Fprintf(os.Stderr, "  screencap -record my-flow https://fleet.example.com\n\n")
	fmt.Fprintf(os.Stderr, "Run a saved workflow:\n")
	fmt.Fprintf(os.Stderr, "  screencap -workflow my-flow https://other-fleet.example.com\n\n")
	fmt.Fprintf(os.Stderr, "List saved workflows:\n")
	fmt.Fprintf(os.Stderr, "  screencap -list\n\n")
	fmt.Fprintf(os.Stderr, "Auth flags (combine with any mode above):\n")
	fmt.Fprintf(os.Stderr, "  -sso                  SSO login (opens browser)\n")
	fmt.Fprintf(os.Stderr, "  -email/-password       email/password login\n")
	fmt.Fprintf(os.Stderr, "  -login                re-authenticate interactively\n")
	fmt.Fprintf(os.Stderr, "  (no auth flag)        reuse saved session\n\n")
	flag.PrintDefaults()
}

func waitForNetworkIdle(ctx context.Context, timeout time.Duration) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		ch := make(chan struct{}, 1)
		lctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		chromedp.ListenTarget(lctx, func(ev interface{}) {
			if e, ok := ev.(*page.EventLifecycleEvent); ok && e.Name == "networkIdle" {
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		})

		select {
		case <-ch:
			return nil
		case <-lctx.Done():
			return nil
		}
	}
}

func pathToFilename(path string) string {
	path = strings.Trim(path, "/")
	return strings.ReplaceAll(path, "/", "-")
}
