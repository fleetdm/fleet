package main

import (
	"context"
	"crypto"
	_ "crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	scepclient "github.com/fleetdm/fleet/v4/server/mdm/scep/client"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/smallstep/scep"
)

// version info
var (
	version = "unknown"
)

const fingerprintHashType = crypto.SHA256

type runCfg struct {
	dir             string
	csrPath         string
	keyPath         string
	keyBits         int
	selfSignPath    string
	certPath        string
	cn              string
	org             string
	ou              string
	locality        string
	province        string
	country         string
	challenge       string
	serverURL       string
	caCertsSelector scep.CertsSelector
	debug           bool
	logfmt          string
	caCertMsg       string
	dnsName         string
}

func run(cfg runCfg) error {
	ctx := context.Background()
	var logger log.Logger
	{
		if strings.ToLower(cfg.logfmt) == "json" {
			logger = log.NewJSONLogger(os.Stderr)
		} else {
			logger = log.NewLogfmtLogger(os.Stderr)
		}
		stdlog.SetOutput(log.NewStdlibAdapter(logger))
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		if !cfg.debug {
			logger = level.NewFilter(logger, level.AllowInfo())
		}
	}
	lginfo := level.Info(logger)

	client, err := scepclient.New(cfg.serverURL, logger, nil)
	if err != nil {
		return err
	}

	key, err := loadOrMakeKey(cfg.keyPath, cfg.keyBits)
	if err != nil {
		return err
	}

	opts := &csrOptions{
		cn:        cfg.cn,
		org:       cfg.org,
		country:   strings.ToUpper(cfg.country),
		ou:        cfg.ou,
		locality:  cfg.locality,
		province:  cfg.province,
		challenge: cfg.challenge,
		key:       key,
		dnsName:   cfg.dnsName,
	}

	csr, err := loadOrMakeCSR(cfg.csrPath, opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var self *x509.Certificate
	cert, err := loadPEMCertFromFile(cfg.certPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		s, err := loadOrSign(cfg.selfSignPath, key, csr)
		if err != nil {
			return err
		}
		self = s
	}

	resp, certNum, err := client.GetCACert(ctx, cfg.caCertMsg)
	if err != nil {
		return err
	}
	var caCerts []*x509.Certificate
	{
		if certNum > 1 {
			caCerts, err = scep.CACerts(resp)
			if err != nil {
				return err
			}
		} else {
			caCerts, err = x509.ParseCertificates(resp)
			if err != nil {
				return err
			}
		}
	}

	if cfg.debug {
		logCerts(level.Debug(logger), caCerts)
	}

	var signerCert *x509.Certificate
	{
		if cert != nil {
			signerCert = cert
		} else {
			signerCert = self
		}
	}

	var msgType scep.MessageType
	{
		// TODO validate CA and set UpdateReq if needed
		if cert != nil {
			msgType = scep.RenewalReq
		} else {
			msgType = scep.PKCSReq
		}
	}

	tmpl := &scep.PKIMessage{
		MessageType: msgType,
		Recipients:  caCerts,
		SignerKey:   key,
		SignerCert:  signerCert,
	}

	if cfg.challenge != "" && msgType == scep.PKCSReq {
		tmpl.CSRReqMessage = &scep.CSRReqMessage{
			ChallengePassword: cfg.challenge,
		}
	}

	msg, err := scep.NewCSRRequest(csr, tmpl, scep.WithLogger(logger), scep.WithCertsSelector(cfg.caCertsSelector))
	if err != nil {
		return errors.Join(err, errors.New("creating csr pkiMessage"))
	}

	var respMsg *scep.PKIMessage

	for {
		// loop in case we get a PENDING response which requires
		// a manual approval.

		respBytes, err := client.PKIOperation(ctx, msg.Raw)
		if err != nil {
			return errors.Join(err, fmt.Errorf("PKIOperation for %s", msgType))
		}

		respMsg, err = scep.ParsePKIMessage(respBytes, scep.WithLogger(logger), scep.WithCACerts(caCerts))
		if err != nil {
			return errors.Join(err, fmt.Errorf("parsing pkiMessage response %s", msgType))
		}

		switch respMsg.PKIStatus {
		case scep.FAILURE:
			return fmt.Errorf("%s request failed, failInfo: %s", msgType, respMsg.FailInfo)
		case scep.PENDING:
			lginfo.Log("pkiStatus", "PENDING", "msg", "sleeping for 30 seconds, then trying again.")
			time.Sleep(30 * time.Second)
			continue
		}
		lginfo.Log("pkiStatus", "SUCCESS", "msg", "server returned a certificate.")
		break // on scep.SUCCESS
	}

	if err := respMsg.DecryptPKIEnvelope(signerCert, key); err != nil {
		return errors.Join(err, fmt.Errorf("decrypt pkiEnvelope, msgType: %s, status %s", msgType, respMsg.PKIStatus))
	}

	respCert := respMsg.CertRepMessage.Certificate
	if err := ioutil.WriteFile(cfg.certPath, pemCert(respCert.Raw), 0o666); err != nil { // nolint:gosec
		return err
	}

	// remove self signer if used
	if self != nil {
		if err := os.Remove(cfg.selfSignPath); err != nil {
			return err
		}
	}

	return nil
}

// logCerts logs the count, number, RDN, and fingerprint of certs to logger
func logCerts(logger log.Logger, certs []*x509.Certificate) {
	logger.Log("msg", "cacertlist", "count", len(certs))
	for i, cert := range certs {
		h := fingerprintHashType.New()
		h.Write(cert.Raw)
		logger.Log(
			"msg", "cacertlist",
			"number", i,
			"rdn", cert.Subject.ToRDNSequence().String(),
			"hash_type", fingerprintHashType.String(),
			"hash", fmt.Sprintf("%x", h.Sum(nil)),
		)
	}
}

// validateFingerprint makes sure fingerprint looks like a hash.
// We remove spaces and colons from fingerprint as it may come in various forms:
//
//	e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
//	E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855
//	e3b0c442 98fc1c14 9afbf4c8 996fb924 27ae41e4 649b934c a495991b 7852b855
//	e3:b0:c4:42:98:fc:1c:14:9a:fb:f4:c8:99:6f:b9:24:27:ae:41:e4:64:9b:93:4c:a4:95:99:1b:78:52:b8:55
func validateFingerprint(fingerprint string) (hash []byte, err error) {
	fingerprint = strings.NewReplacer(" ", "", ":", "").Replace(fingerprint)
	hash, err = hex.DecodeString(fingerprint)
	if err != nil {
		return
	}
	if len(hash) != fingerprintHashType.Size() {
		err = fmt.Errorf("invalid %s hash length", fingerprintHashType)
	}
	return
}

func validateFlags(keyPath, serverURL, caFingerprint string, useKeyEnciphermentSelector bool) error {
	if keyPath == "" {
		return errors.New("must specify private key path")
	}
	if serverURL == "" {
		return errors.New("must specify server-url flag parameter")
	}
	_, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server-url flag parameter %s", err)
	}
	if caFingerprint != "" && useKeyEnciphermentSelector {
		return errors.New("ca-fingerprint and key-encipherment-selector can't be used at the same time")
	}
	return nil
}

func main() {
	var (
		flVersion           = flag.Bool("version", false, "prints version information")
		flServerURL         = flag.String("server-url", "", "SCEP server url")
		flChallengePassword = flag.String("challenge", "", "enforce a challenge password")
		flPKeyPath          = flag.String("private-key", "", "private key path, if there is no key, scepclient will create one")
		flCertPath          = flag.String("certificate", "", "certificate path, if there is no key, scepclient will create one")
		flKeySize           = flag.Int("keySize", 2048, "rsa key size")
		flOrg               = flag.String("organization", "scep-client", "organization for cert")
		flCName             = flag.String("cn", "scepclient", "common name for certificate")
		flOU                = flag.String("ou", "MDM", "organizational unit for certificate")
		flLoc               = flag.String("locality", "", "locality for certificate")
		flProvince          = flag.String("province", "", "province for certificate")
		flCountry           = flag.String("country", "US", "country code in certificate")
		flCACertMessage     = flag.String("cacert-message", "", "message sent with GetCACert operation")
		flDNSName           = flag.String("dnsname", "", "DNS name to be included in the certificate (SAN)")

		// in case of multiple certificate authorities, we need to figure out who the recipient of the encrypted
		// data is. This can be done using either the CA fingerprint, or based on the key usage encoded in the
		// certificates returned by the authority.
		flCAFingerprint = flag.String("ca-fingerprint", "",
			"SHA-256 digest of CA certificate for NDES server. Note: Changed from MD5.")
		flKeyEnciphermentSelector = flag.Bool("key-encipherment-selector", false, "Filter CA certificates by key encipherment usage")

		flDebugLogging = flag.Bool("debug", false, "enable debug logging")
		flLogJSON      = flag.Bool("log-json", false, "use JSON for log output")
	)

	flag.Parse()

	// print version information
	if *flVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if err := validateFlags(*flPKeyPath, *flServerURL, *flCAFingerprint, *flKeyEnciphermentSelector); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	caCertsSelector := scep.NopCertsSelector()
	switch {
	case *flCAFingerprint != "":
		hash, err := validateFingerprint(*flCAFingerprint)
		if err != nil {
			fmt.Printf("invalid fingerprint: %s\n", err)
			os.Exit(1)
		}
		caCertsSelector = scep.FingerprintCertsSelector(fingerprintHashType, hash)
	case *flKeyEnciphermentSelector:
		caCertsSelector = scep.EnciphermentCertsSelector()
	}

	dir := filepath.Dir(*flPKeyPath)
	csrPath := dir + "/csr.pem"
	selfSignPath := dir + "/self.pem"
	if *flCertPath == "" {
		*flCertPath = dir + "/client.pem"
	}
	var logfmt string
	if *flLogJSON {
		logfmt = "json"
	}

	cfg := runCfg{
		dir:             dir,
		csrPath:         csrPath,
		keyPath:         *flPKeyPath,
		keyBits:         *flKeySize,
		selfSignPath:    selfSignPath,
		certPath:        *flCertPath,
		cn:              *flCName,
		org:             *flOrg,
		country:         *flCountry,
		locality:        *flLoc,
		ou:              *flOU,
		province:        *flProvince,
		challenge:       *flChallengePassword,
		serverURL:       *flServerURL,
		caCertsSelector: caCertsSelector,
		debug:           *flDebugLogging,
		logfmt:          logfmt,
		caCertMsg:       *flCACertMessage,
		dnsName:         *flDNSName,
	}

	if err := run(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
