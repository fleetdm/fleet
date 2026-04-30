package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// wizardModel runs a one-time setup conversation: ngrok static domain, MDM
// server private key (only when missing), premium on/off. Outputs are written
// back into the root model's Config + private key file before the TUI moves
// to the dashboard.
type wizardModel struct {
	step       wizardStep
	ngrokInput textinput.Model
	mdmInput   textinput.Model
	premium    bool

	// askMDMKey is decided at construction time: only ask if there's no
	// key on disk yet. Skipping the field when we have one keeps reruns
	// fast and avoids tempting the user to paste it again.
	askMDMKey bool

	// done marks the wizard finished — the root model reads this to know
	// when to advance the screen and persist results.
	done bool
}

type wizardStep int

const (
	wizStepNgrok wizardStep = iota
	wizStepMDMKey
	wizStepPremium
)

func newWizardModel(initial Config, hasPrivateKey bool) wizardModel {
	ngrok := textinput.New()
	ngrok.Placeholder = "fleet-pm-jane.ngrok-free.app"
	ngrok.SetValue(initial.Ngrok.StaticDomain)
	ngrok.CharLimit = 200
	ngrok.Width = 50
	ngrok.Focus()

	mdm := textinput.New()
	mdm.Placeholder = "paste your 32+ character key"
	mdm.CharLimit = 1024
	mdm.Width = 50
	mdm.EchoMode = textinput.EchoPassword
	mdm.EchoCharacter = '•'

	return wizardModel{
		step:       wizStepNgrok,
		ngrokInput: ngrok,
		mdmInput:   mdm,
		premium:    initial.Fleet.Premium,
		askMDMKey:  !hasPrivateKey,
	}
}

func (w wizardModel) update(msg tea.Msg) (wizardModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			return w.advance()
		case "esc":
			return w.back()
		case "left", "h":
			if w.step == wizStepPremium {
				w.premium = true
			}
			return w, nil
		case "right", "l":
			if w.step == wizStepPremium {
				w.premium = false
			}
			return w, nil
		case " ":
			// space toggles the premium choice — common ergonomic shortcut.
			if w.step == wizStepPremium {
				w.premium = !w.premium
				return w, nil
			}
		}
	}

	var cmd tea.Cmd
	switch w.step {
	case wizStepNgrok:
		w.ngrokInput, cmd = w.ngrokInput.Update(msg)
	case wizStepMDMKey:
		w.mdmInput, cmd = w.mdmInput.Update(msg)
	}
	return w, cmd
}

// advance moves to the next step, or marks the wizard done if we're on the
// last one. Skips the MDM key step when a key is already on disk.
func (w wizardModel) advance() (wizardModel, tea.Cmd) {
	switch w.step {
	case wizStepNgrok:
		if strings.TrimSpace(w.ngrokInput.Value()) == "" {
			return w, nil // require a domain — nothing else makes sense without it
		}
		if w.askMDMKey {
			w.step = wizStepMDMKey
			w.ngrokInput.Blur()
			w.mdmInput.Focus()
		} else {
			w.step = wizStepPremium
			w.ngrokInput.Blur()
		}
	case wizStepMDMKey:
		if strings.TrimSpace(w.mdmInput.Value()) == "" {
			return w, nil
		}
		w.step = wizStepPremium
		w.mdmInput.Blur()
	case wizStepPremium:
		w.done = true
	}
	return w, nil
}

func (w wizardModel) back() (wizardModel, tea.Cmd) {
	switch w.step {
	case wizStepMDMKey:
		w.step = wizStepNgrok
		w.mdmInput.Blur()
		w.ngrokInput.Focus()
	case wizStepPremium:
		if w.askMDMKey {
			w.step = wizStepMDMKey
			w.mdmInput.Focus()
		} else {
			w.step = wizStepNgrok
			w.ngrokInput.Focus()
		}
	}
	return w, nil
}

// applyTo merges wizard answers into the supplied Config and returns it. The
// MDM key (if asked for) is returned separately so the caller can persist it
// to ~/.config/fleet-ship/server_private_key with the right file mode.
func (w wizardModel) applyTo(cfg Config) (Config, string) {
	cfg.Ngrok.StaticDomain = strings.TrimSpace(w.ngrokInput.Value())
	cfg.Fleet.Premium = w.premium
	if cfg.Fleet.Port == 0 {
		cfg.Fleet.Port = 8080
	}
	mdm := ""
	if w.askMDMKey {
		mdm = strings.TrimSpace(w.mdmInput.Value())
	}
	return cfg, mdm
}

func (w wizardModel) view(width int) string {
	if width <= 0 {
		width = 80
	}

	header := styleHeaderBrand.Render("Fleet ship") + styleHint.Render("  ·  setup")

	var body string
	switch w.step {
	case wizStepNgrok:
		body = renderField(
			"ngrok static domain",
			"Get one for free at "+styleURL.Render("https://dashboard.ngrok.com/domains"),
			w.ngrokInput.View(),
		)
	case wizStepMDMKey:
		body = renderField(
			"MDM server private key",
			"Paste from your 1Password Fleet dev login. Stored at "+
				styleURL.Render("~/.config/fleet-ship/server_private_key"),
			w.mdmInput.View(),
		)
	case wizStepPremium:
		body = renderPremiumChoice(w.premium)
	}

	hints := strings.Join([]string{
		styleKey.Render("enter") + " " + styleHint.Render("next"),
		styleKey.Render("esc") + " " + styleHint.Render("back"),
		styleKey.Render("q") + " " + styleHint.Render("quit"),
	}, styleHint.Render("   ·   "))

	progress := styleHint.Render(stepLabel(w))

	return stylePane.Width(width - 2).Render(strings.Join([]string{
		header,
		"",
		progress,
		"",
		body,
		"",
		hints,
	}, "\n"))
}

func renderField(label, helper, input string) string {
	return strings.Join([]string{
		styleLabel.Render(label),
		styleHint.Render(helper),
		"",
		"  " + input,
	}, "\n")
}

func renderPremiumChoice(premium bool) string {
	on := " ( ) free"
	off := " ( ) free"
	if premium {
		on = " (•) " + styleOK().Render("premium")
	} else {
		off = " (•) " + styleOK().Render("free")
	}
	left := on
	right := off
	if !premium {
		left = " ( ) " + styleHint.Render("premium")
	} else {
		right = " ( ) " + styleHint.Render("free")
	}
	return strings.Join([]string{
		styleLabel.Render("Fleet edition"),
		styleHint.Render("Premium enables paid features (--dev_license). Most PMs want this on."),
		"",
		"  " + left + "    " + right,
		"",
		styleHint.Render("←/→ or space toggles. enter confirms."),
	}, "\n")
}

// stepLabel renders something like "Step 2 of 3" — the count varies based on
// whether the MDM key step is being skipped.
func stepLabel(w wizardModel) string {
	total := 2
	if w.askMDMKey {
		total = 3
	}
	current := 1
	switch w.step {
	case wizStepMDMKey:
		current = 2
	case wizStepPremium:
		if w.askMDMKey {
			current = 3
		} else {
			current = 2
		}
	}
	return "Step " + itoa(current) + " of " + itoa(total)
}

// itoa is just strconv.Itoa, inlined to keep this file self-contained.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
