package luks_runner

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
)

type KeyEscrower interface {
	SendLinuxKeyEscrowResponse(LuksResponse) error
}

type LuksRunner struct {
	escrower KeyEscrower
	notifier dialog.Dialog
}

type LuksResponse struct {
	Key string `json:"key"`
	Err string `json:"err"`
}

func New(escrower KeyEscrower, notifier dialog.Dialog) *LuksRunner {
	return &LuksRunner{
		escrower: escrower,
		notifier: notifier,
	}
}
