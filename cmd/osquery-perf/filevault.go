package main

import (
	"crypto/x509"
	"encoding/base64"
	"log"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
)

// A real Mac that installs Fleet's FileVault profile encrypts its personal recovery key
// with the certificate the profile's FDERecoveryKeyEscrow payload points to, and fleetd
// reports the result through the filevault_prk table. Simulating that gives Fleet keys it
// can actually decrypt, so its key verification and FileVault status paths run at scale.

const (
	fileVaultEscrowPayloadType = "com.apple.security.FDERecoveryKeyEscrow"
	fileVaultCertPayloadType   = "com.apple.security.pkcs1"
)

// fileVaultEscrowCert returns the certificate the profile's FDERecoveryKeyEscrow payload
// encrypts the recovery key with, or nil if the profile has no escrow payload.
func fileVaultEscrowCert(profile []byte) *x509.Certificate {
	var parsed struct {
		PayloadContent []map[string]any `plist:"PayloadContent"`
	}
	if err := plist.Unmarshal(profile, &parsed); err != nil {
		return nil
	}
	var certUUID string
	for _, p := range parsed.PayloadContent {
		if p["PayloadType"] == fileVaultEscrowPayloadType {
			certUUID, _ = p["EncryptCertPayloadUUID"].(string)
		}
	}
	if certUUID == "" {
		return nil
	}
	for _, p := range parsed.PayloadContent {
		if p["PayloadType"] != fileVaultCertPayloadType || p["PayloadUUID"] != certUUID {
			continue
		}
		der, ok := p["PayloadContent"].([]byte)
		if !ok {
			return nil
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil
		}
		return cert
	}
	return nil
}

// escrowFileVaultKey records an encrypted recovery key if cmd installs a profile with a
// FileVault escrow payload. The key itself is seed-derived, like a Mac's PRK that survives
// a restart of the agent.
func (a *agent) escrowFileVaultKey(cmd *mdm.Command) {
	profile := installProfilePayload(cmd)
	if profile == nil {
		return
	}
	cert := fileVaultEscrowCert(profile)
	if cert == nil {
		return
	}
	prk := a.id.str("filevault-prk", serialCharset, 24)
	prk = prk[0:4] + "-" + prk[4:8] + "-" + prk[8:12] + "-" + prk[12:16] + "-" + prk[16:20] + "-" + prk[20:24]
	enveloped, err := pkcs7.Encrypt([]byte(prk), []*x509.Certificate{cert})
	if err != nil {
		log.Printf("agent %d: encrypting FileVault recovery key: %s", a.agentIndex, err)
		return
	}
	key := base64.StdEncoding.EncodeToString(enveloped)
	a.fileVaultKey.Store(&key)
}
