package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// ddmMethods is a helper struct to abstract method calls into their respective user/device methods.
type ddmMethods struct {
	DeclarativeManagement func(endpoint string, data ...fleet.MDMAppleDDMStatusReport) (*http.Response, error)

	IncrementTokensErrors            func()
	IncrementTokensSuccess           func()
	IncrementDeclarationItemsErrors  func()
	IncrementDeclarationItemsSuccess func()

	IncrementConfigurationErrors  func()
	IncrementConfigurationSuccess func()

	IncrementActivationErrors  func()
	IncrementActivationSuccess func()

	IncrementAssetErrors  func()
	IncrementAssetSuccess func()

	IncrementStatusErrors  func()
	IncrementStatusSuccess func()

	getGlobalToken func() string
	setGlobalToken func(token string)

	getDeclTokens func() map[string]string
	setDeclTokens func(tokens map[string]string)
}

func (a *agent) doDeclarativeManagement(cmd *mdm.Command, methods ddmMethods) {
	const maxAttempts = 3

	// prevToken starts as the last-applied global token. On each iteration,
	// a tokens fetch is compared to it: if it matches, the server has settled
	// (or nothing changed on the first pass). If it differs, we sync
	// declaration-items and fetch changed declarations, then loop to check
	// again (mimicking the real device behavior, see
	// https://github.com/fleetdm/fleet/issues/43050#issuecomment-4252241277).
	prevToken := methods.getGlobalToken()
	var items *fleet.MDMAppleDDMDeclarationItemsResponse
	var currentTokens map[string]string
	changed := false

	for range maxAttempts {
		// Fetch tokens — on the first pass this is the initial check against
		// the cached token; on subsequent passes it is the convergence check
		// for the previous iteration's sync.
		globalToken, err := a.ddmFetchTokens(methods)
		if err != nil {
			return
		}
		if globalToken == prevToken {
			break // nothing changed, or server has settled
		}

		// Fetch declaration-items manifest
		items, err = a.ddmFetchDeclarationItems(methods)
		if err != nil {
			return
		}

		// Check each manifest item against cached tokens, fetch changed ones.
		currentTokens = make(map[string]string, len(items.Declarations.Activations)+len(items.Declarations.Configurations)+len(items.Declarations.Assets))
		for _, d := range items.Declarations.Activations {
			currentTokens[d.Identifier] = d.ServerToken
			if methods.getDeclTokens()[d.Identifier] != d.ServerToken {
				if err := a.ddmFetchDeclaration("activation", d.Identifier, methods); err != nil {
					return
				}
				changed = true
			}
		}

		for _, d := range items.Declarations.Assets {
			currentTokens[d.Identifier] = d.ServerToken
			if methods.getDeclTokens()[d.Identifier] != d.ServerToken {
				if err := a.ddmFetchDeclaration("asset", d.Identifier, methods); err != nil {
					return
				}
				changed = true
			}
		}

		for _, d := range items.Declarations.Configurations {
			currentTokens[d.Identifier] = d.ServerToken
			if methods.getDeclTokens()[d.Identifier] != d.ServerToken {
				if err := a.ddmFetchDeclaration("configuration", d.Identifier, methods); err != nil {
					return
				}
				changed = true
			}
		}

		// Check for removed items (in cache but not in manifest) - no need to check if changes are
		// already detected, as the whole set of declaration tokens will get replaced with the current ones.
		// This is just to detect the case where the only change is a removal.
		if !changed {
			for id := range methods.getDeclTokens() {
				if _, ok := currentTokens[id]; !ok {
					changed = true
					break
				}
			}
		}

		prevToken = globalToken
	}

	if !changed || items == nil {
		return
	}

	// Server has settled (or max attempts exhausted) and declarations
	// changed — send a single consolidated status report and update cache.
	if err := a.ddmSendStatus(items, methods); err != nil {
		return
	}
	methods.setGlobalToken(prevToken)
	methods.setDeclTokens(currentTokens)
}

func (a *agent) ddmFetchTokens(methods ddmMethods) (string, error) {
	r, err := methods.DeclarativeManagement("tokens")
	if err != nil {
		log.Printf("DDM tokens request failed: %s", err)
		methods.IncrementTokensErrors()
		return "", err
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("DDM tokens read body failed: %s", err)
		methods.IncrementTokensErrors()
		return "", err
	}
	var resp fleet.MDMAppleDDMTokensResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Printf("DDM tokens unmarshal failed: %s", err)
		methods.IncrementTokensErrors()
		return "", err
	}
	methods.IncrementTokensSuccess()
	return resp.SyncTokens.DeclarationsToken, nil
}

func (a *agent) ddmFetchDeclarationItems(methods ddmMethods) (*fleet.MDMAppleDDMDeclarationItemsResponse, error) {
	r, err := methods.DeclarativeManagement("declaration-items")
	if err != nil {
		log.Printf("DDM declaration-items request failed: %s", err)
		methods.IncrementDeclarationItemsErrors()
		return nil, err
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("DDM declaration-items read body failed: %s", err)
		methods.IncrementDeclarationItemsErrors()
		return nil, err
	}
	var items fleet.MDMAppleDDMDeclarationItemsResponse
	if err := json.Unmarshal(body, &items); err != nil {
		log.Printf("DDM declaration-items unmarshal failed: %s", err)
		methods.IncrementDeclarationItemsErrors()
		return nil, err
	}
	methods.IncrementDeclarationItemsSuccess()
	return &items, nil
}

func (a *agent) ddmFetchDeclaration(kind, identifier string, methods ddmMethods) error {
	path := fmt.Sprintf("declaration/%s/%s", kind, identifier)
	r, err := methods.DeclarativeManagement(path)
	if err != nil {
		log.Printf("DDM %s request failed: %s", path, err)
		a.ddmIncrementDeclError(kind, methods)
		return err
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("DDM %s read body failed: %s", path, err)
		a.ddmIncrementDeclError(kind, methods)
		return err
	}
	switch kind {
	case "activation":
		var act fleet.MDMAppleDDMActivation
		if err := json.Unmarshal(body, &act); err != nil {
			log.Printf("DDM %s unmarshal failed: %s", path, err)
			a.ddmIncrementDeclError(kind, methods)
			return err
		}
		methods.IncrementActivationSuccess()
	case "configuration":
		var decl fleet.MDMAppleDeclaration
		if err := json.Unmarshal(body, &decl); err != nil {
			log.Printf("DDM %s unmarshal failed: %s", path, err)
			a.ddmIncrementDeclError(kind, methods)
			return err
		}
		methods.IncrementConfigurationSuccess()
	case "asset":
		var asset fleet.RawDDMAsset
		if err := json.Unmarshal(body, &asset); err != nil {
			log.Printf("DDM %s unmarshal failed: %s", path, err)
			a.ddmIncrementDeclError(kind, methods)
			return err
		}
		methods.IncrementAssetSuccess()
	}
	return nil
}

func (a *agent) ddmIncrementDeclError(kind string, methods ddmMethods) {
	switch kind {
	case "activation":
		methods.IncrementActivationErrors()
	case "configuration":
		methods.IncrementConfigurationErrors()
	case "asset":
		methods.IncrementAssetErrors()
	}
}

func (a *agent) ddmSendStatus(items *fleet.MDMAppleDDMDeclarationItemsResponse, methods ddmMethods) error {
	report := fleet.MDMAppleDDMStatusReport{}
	for _, d := range items.Declarations.Activations {
		report.StatusItems.Management.Declarations.Activations = append(
			report.StatusItems.Management.Declarations.Activations,
			fleet.MDMAppleDDMStatusDeclaration{
				Active: true, Valid: fleet.MDMAppleDeclarationValid,
				Identifier: d.Identifier, ServerToken: d.ServerToken,
			},
		)
	}
	for _, d := range items.Declarations.Assets {
		report.StatusItems.Management.Declarations.Assets = append(
			report.StatusItems.Management.Declarations.Assets,
			fleet.MDMAppleDDMStatusDeclaration{
				Active: true, Valid: fleet.MDMAppleDeclarationValid,
				Identifier: d.Identifier, ServerToken: d.ServerToken,
			},
		)
	}
	for _, d := range items.Declarations.Configurations {
		report.StatusItems.Management.Declarations.Configurations = append(
			report.StatusItems.Management.Declarations.Configurations,
			fleet.MDMAppleDDMStatusDeclaration{
				Active: true, Valid: fleet.MDMAppleDeclarationValid,
				Identifier: d.Identifier, ServerToken: d.ServerToken,
			},
		)
	}

	r, err := a.macMDMClient.DeclarativeManagement("status", report)
	if err != nil {
		log.Printf("DDM status request failed: %s", err)
		methods.IncrementStatusErrors()
		return err
	}
	defer r.Body.Close()
	_, _ = io.Copy(io.Discard, r.Body)
	if r.StatusCode != http.StatusOK {
		log.Printf("DDM status response unexpected: %d", r.StatusCode)
		methods.IncrementStatusErrors()
		return fmt.Errorf("unexpected status code: %d", r.StatusCode)
	}
	methods.IncrementStatusSuccess()
	return nil
}
