//go:build linux

package securehw

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	keyfile "github.com/foxboron/go-tpm-keyfiles"
	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpm2/transport/linuxtpm"
	"github.com/rs/zerolog"
)

// tpm2SecureHW implements the SecureHW interface using TPM 2.0.
type tpm2SecureHW struct {
	device transport.TPMCloser

	logger      zerolog.Logger
	keyFilePath string
}

const tpm20DevicePath = "/dev/tpmrm0"

// Creates a new SecureHW instance using TPM 2.0.
// It attempts to open the TPM device using the provided configuration.
func newSecureHW(metadataDir string, logger zerolog.Logger) (SecureHW, error) {
	if metadataDir == "" {
		return nil, errors.New("required metadata directory not set")
	}

	logger.Info().Msg("initializing TPM 2.0 connection")

	// Open the TPM 2.0 resource manager, which
	// - Provides managed access to TPM resources, allowing multiple applications to share the TPM safely.
	// - Used by the TPM2 Access Broker and Resource Manager (tpm2-abrmd or the kernel resource manager).
	device, err := linuxtpm.Open(tpm20DevicePath)
	if err != nil {
		return nil, ErrSecureHWUnavailable{
			Message: fmt.Sprintf("failed to open TPM 2.0 device %q: %s", tpm20DevicePath, err.Error()),
		}
	}

	logger.Info().Str("device_path", tpm20DevicePath).Msg("successfully opened TPM 2.0 device")

	return &tpm2SecureHW{
		device: device,

		logger:      zerolog.Nop(),
		keyFilePath: filepath.Join(metadataDir, "host_identity_tpm.pem"),
	}, nil
}

// CreateKey partially implements SecureHW.
func (t *tpm2SecureHW) CreateKey() (Key, error) {
	t.logger.Info().Msg("creating new ECC key in TPM")

	parentKeyHandle, err := t.createParentKey()
	if err != nil {
		return nil, fmt.Errorf("get or create TPM parent key: %w", err)
	}

	curveID, curveName := t.selectBestECCCurve()
	t.logger.Info().Str("curve", curveName).Msg("selected ECC curve for key creation")

	// Create an ECC key template for the child key
	t.logger.Debug().Str("curve", curveName).Msg("creating ECC key template")
	eccTemplate := tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			UserWithAuth:        true, // Required even if password is nil
			SignEncrypt:         true,
			// We will just use this child key for signing.
			// If we need encryption in the future we can create a separate key for it.
			// It's usually recommended to have separate keys for signing and encryption.
			Decrypt: false,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgECC,
			&tpm2.TPMSECCParms{
				CurveID: curveID,
			},
		),
	})

	// Create the key under the transient parent
	t.logger.Debug().Msg("creating child key")
	createKey, err := tpm2.Create{
		ParentHandle: parentKeyHandle,
		InPublic:     eccTemplate,
	}.Execute(t.device)
	if err != nil {
		return nil, fmt.Errorf("create child key: %w", err)
	}

	t.logger.Debug().Msg("Loading created key")
	loadedKey, err := tpm2.Load{
		ParentHandle: parentKeyHandle,
		InPrivate:    createKey.OutPrivate,
		InPublic:     createKey.OutPublic,
	}.Execute(t.device)
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}

	t.logger.Debug().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("key loaded successfully")

	cleanUpOnError := func() {
		flush := tpm2.FlushContext{
			FlushHandle: loadedKey.ObjectHandle,
		}
		_, _ = flush.Execute(t.device)
	}

	if err != nil {
		cleanUpOnError()
		return nil, fmt.Errorf("save key context: %w", err)
	}

	t.logger.Info().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("key created and context saved successfully")

	if err := t.saveTPMKeyFile(createKey.OutPrivate, createKey.OutPublic); err != nil {
		cleanUpOnError()
		return nil, fmt.Errorf("write TPM keyfile to file: %w", err)
	}

	// Create and return the key
	return &tpm2Key{
		tpm:    t.device,
		handle: tpm2.NamedHandle{Handle: loadedKey.ObjectHandle, Name: loadedKey.Name},
		public: createKey.OutPublic,
		logger: t.logger,
	}, nil
}

// createParentKey creates a transient Storage Root Key for use as a parent key
//
// NOTE: It creates the parent key deterministically so this can be called when loading a child key.
func (t *tpm2SecureHW) createParentKey() (tpm2.NamedHandle, error) {
	t.logger.Debug().Msg("creating transient RSA 2048-bit parent key")

	// Create a parent key template with required attributes
	parentTemplate := tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgRSA,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true, // bound to TPM that created it
			FixedParent:         true, // Required, based on manual testing
			SensitiveDataOrigin: true, // key material generated internally
			UserWithAuth:        true, // Required, even if we use nil password
			Decrypt:             true, // Allows key to be used for decryption/unwrapping
			Restricted:          true, // Limits use to decryption of child keys
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgRSA,
			&tpm2.TPMSRSAParms{
				KeyBits: 2048,
				Symmetric: tpm2.TPMTSymDefObject{
					Algorithm: tpm2.TPMAlgAES,
					KeyBits: tpm2.NewTPMUSymKeyBits(
						tpm2.TPMAlgAES,
						tpm2.TPMKeyBits(128),
					),
					Mode: tpm2.NewTPMUSymMode(
						tpm2.TPMAlgAES,
						tpm2.TPMAlgCFB,
					),
				},
			},
		),
	})

	// If this command is called multiple times with the same inPublic parameter,
	// inSensitive.data, and Primary Seed, the TPM shall produce the same Primary Object.
	primaryKey, err := tpm2.CreatePrimary{
		PrimaryHandle: tpm2.TPMRHOwner,
		InPublic:      parentTemplate,
	}.Execute(t.device)
	if err != nil {
		return tpm2.NamedHandle{}, fmt.Errorf("create transient parent key: %w", err)
	}

	t.logger.Info().
		Str("handle", fmt.Sprintf("0x%x", primaryKey.ObjectHandle)).
		Msg("created transient parent key successfully")

	// Return the transient key as a NamedHandle
	return tpm2.NamedHandle{
		Handle: primaryKey.ObjectHandle,
		Name:   primaryKey.Name,
	}, nil
}

// selectBestECCCurve checks if the TPM supports ECC P-384, otherwise returns P-256
func (t *tpm2SecureHW) selectBestECCCurve() (tpm2.TPMECCCurve, string) {
	t.logger.Debug().Msg("checking TPM ECC curve support")

	// Try to create a test key with P-384 to check support
	// This is a more reliable method than querying capabilities
	testTemplate := tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			UserWithAuth:        true, // Required even if password is nil
			SensitiveDataOrigin: true,
			SignEncrypt:         true,
			Decrypt:             true,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgECC,
			&tpm2.TPMSECCParms{
				CurveID: tpm2.TPMECCNistP384,
			},
		),
	})

	// Try to create a primary key with P-384 to test support
	testKey, err := tpm2.CreatePrimary{
		PrimaryHandle: tpm2.TPMRHOwner,
		InPublic:      testTemplate,
	}.Execute(t.device)
	if err != nil {
		t.logger.Debug().Err(err).Msg("TPM does not support P-384, using P-256")
		return tpm2.TPMECCNistP256, "P-256"
	}

	// Clean up the test key
	flush := tpm2.FlushContext{
		FlushHandle: testKey.ObjectHandle,
	}
	_, _ = flush.Execute(t.device)

	t.logger.Debug().Msg("TPM supports P-384")
	return tpm2.TPMECCNistP384, "P-384"
}

func (t *tpm2SecureHW) saveTPMKeyFile(privateKey tpm2.TPM2BPrivate, publicKey tpm2.TPM2BPublic) error {
	k := keyfile.NewTPMKey(
		keyfile.OIDOldLoadableKey,
		publicKey,
		privateKey,
		keyfile.WithDescription("fleetd httpsig key"),
	)
	if err := os.WriteFile(t.keyFilePath, k.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to save keyfile: %w", err)
	}
	return nil
}

func (t *tpm2SecureHW) loadTPMKeyFile() (privateKey *tpm2.TPM2BPrivate, publicKey *tpm2.TPM2BPublic, err error) {
	keyfileBytes, err := os.ReadFile(t.keyFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, ErrKeyNotFound{}
		}
		return nil, nil, fmt.Errorf("failed to read keyfile path: %w", err)
	}
	k, err := keyfile.Decode(keyfileBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode keyfile: %w", err)
	}
	return &k.Privkey, &k.Pubkey, nil
}

// LoadKey partially implements SecureHW.
func (t *tpm2SecureHW) LoadKey() (Key, error) {
	private, public, err := t.loadTPMKeyFile()
	if err != nil {
		return nil, err
	}

	// Get the parent key handle.
	//
	// NOTE: createParentKey calls CreatePrimary which creates the parent key
	// deterministically so this can be called when loadind a child key.
	parentKeyHandle, err := t.createParentKey()
	if err != nil {
		return nil, fmt.Errorf("get parent key: %w", err)
	}

	// Load the key using the parent handle.
	t.logger.Debug().Uint32("parent_handle", uint32(parentKeyHandle.Handle)).Msg("loading parent key")
	loadedKey, err := tpm2.Load{
		ParentHandle: parentKeyHandle,
		InPrivate:    *private,
		InPublic:     *public,
	}.Execute(t.device)
	if err != nil {
		return nil, fmt.Errorf("load parent key: %w", err)
	}

	t.logger.Debug().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("key loaded successfully")

	t.logger.Info().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("key loaded successfully")

	return &tpm2Key{
		tpm: t.device,
		handle: tpm2.NamedHandle{
			Handle: loadedKey.ObjectHandle,
			Name:   loadedKey.Name,
		},
		public: *public,
		logger: t.logger,
	}, nil
}

// Close partially implements SecureHW.
func (t *tpm2SecureHW) Close() error {
	t.logger.Info().Msg("closing TPM device")
	if t.device != nil {
		err := t.device.Close()
		if err != nil {
			t.logger.Error().Err(err).Msg("error closing TPM device")
			return err
		}
		t.device = nil
		t.logger.Debug().Msg("TPM device closed successfully")
	}
	return nil
}

// tpm2Key implements the Key interface using TPM 2.0.
type tpm2Key struct {
	tpm    transport.TPMCloser
	handle tpm2.NamedHandle
	public tpm2.TPM2BPublic
	logger zerolog.Logger
}

func (k *tpm2Key) Signer() (crypto.Signer, error) {
	signer, _, err := k.createSigner(false)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

func (k *tpm2Key) HTTPSigner() (HTTPSigner, error) {
	signer, algo, err := k.createSigner(true)
	if err != nil {
		return nil, err
	}
	return &httpSigner{
		Signer: signer,
		algo:   algo,
	}, nil
}

type httpSigner struct {
	crypto.Signer
	algo ECCAlgorithm
}

func (h *httpSigner) ECCAlgorithm() ECCAlgorithm {
	return h.algo
}

func (k *tpm2Key) createSigner(httpsign bool) (s crypto.Signer, algo ECCAlgorithm, err error) {
	pub, err := k.public.Contents()
	if err != nil {
		return nil, 0, fmt.Errorf("get public key contents: %w", err)
	}

	if pub.Type != tpm2.TPMAlgECC {
		return nil, 0, errors.New("not an ECC key")
	}

	eccDetail, err := pub.Parameters.ECCDetail()
	if err != nil {
		return nil, 0, fmt.Errorf("get ECC details: %w", err)
	}

	eccUnique, err := pub.Unique.ECC()
	if err != nil {
		return nil, 0, fmt.Errorf("get ECC unique: %w", err)
	}

	// Create crypto.PublicKey based on curve
	var publicKey *ecdsa.PublicKey
	switch eccDetail.CurveID {
	case tpm2.TPMECCNistP256:
		publicKey = &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(eccUnique.X.Buffer),
			Y:     new(big.Int).SetBytes(eccUnique.Y.Buffer),
		}
		algo = ECCAlgorithmP256
	case tpm2.TPMECCNistP384:
		publicKey = &ecdsa.PublicKey{
			Curve: elliptic.P384(),
			X:     new(big.Int).SetBytes(eccUnique.X.Buffer),
			Y:     new(big.Int).SetBytes(eccUnique.Y.Buffer),
		}
		algo = ECCAlgorithmP384
	default:
		return nil, 0, fmt.Errorf("unsupported ECC curve: %v", eccDetail.CurveID)
	}

	return &tpm2Signer{
		tpm:       k.tpm,
		handle:    k.handle,
		publicKey: publicKey,
		httpsign:  httpsign,
	}, algo, nil
}

func (k *tpm2Key) Public() (crypto.PublicKey, error) {
	signer, err := k.Signer()
	if err != nil {
		return nil, err
	}
	return signer.Public(), nil
}

func (k *tpm2Key) Close() error {
	if k.handle.Handle != 0 {
		flush := tpm2.FlushContext{
			FlushHandle: k.handle.Handle,
		}
		_, err := flush.Execute(k.tpm)
		k.handle = tpm2.NamedHandle{}
		return err
	}
	return nil
}

// tpm2Signer implements crypto.Signer using TPM 2.0.
type tpm2Signer struct {
	tpm       transport.TPM
	handle    tpm2.NamedHandle
	publicKey *ecdsa.PublicKey
	httpsign  bool // true for RFC 9421-compatible HTTP signatures, false for standard ECDSA
}

// _ ensures tpm2Signer satisfies the crypto.Signer interface at compile time.
var _ crypto.Signer = (*tpm2Signer)(nil)

func (s *tpm2Signer) Public() crypto.PublicKey {
	return s.publicKey
}

func (s *tpm2Signer) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// Determine hash algorithm
	var hashAlg tpm2.TPMAlgID
	switch opts.HashFunc() {
	case crypto.SHA256:
		hashAlg = tpm2.TPMAlgSHA256
	case crypto.SHA384:
		hashAlg = tpm2.TPMAlgSHA384
	default:
		return nil, fmt.Errorf("unsupported hash function: %v", opts.HashFunc())
	}

	// Sign with TPM using ECDSA.
	// ECC keys are used with ECDSA (Elliptic Curve Digital Signature Algorithm) for signing
	sign := tpm2.Sign{
		KeyHandle: s.handle,
		Digest: tpm2.TPM2BDigest{
			Buffer: digest,
		},
		InScheme: tpm2.TPMTSigScheme{
			Scheme: tpm2.TPMAlgECDSA,
			Details: tpm2.NewTPMUSigScheme(
				tpm2.TPMAlgECDSA,
				&tpm2.TPMSSchemeHash{
					HashAlg: hashAlg,
				},
			),
		},
		Validation: tpm2.TPMTTKHashCheck{
			Tag: tpm2.TPMSTHashCheck,
		},
	}

	rsp, err := sign.Execute(s.tpm)
	if err != nil {
		return nil, fmt.Errorf("TPM sign: %w", err)
	}

	// Check signature type and extract ECDSA signature
	if rsp.Signature.SigAlg != tpm2.TPMAlgECDSA {
		return nil, fmt.Errorf("unexpected signature algorithm: %v", rsp.Signature.SigAlg)
	}

	// Get the ECDSA signature
	ecdsaSig, err := rsp.Signature.Signature.ECDSA()
	if err != nil {
		return nil, fmt.Errorf("get ECDSA signature: %w", err)
	}

	if s.httpsign {
		// RFC 9421-compatible HTTP signature format: fixed-width r||s
		curveBits := s.publicKey.Curve.Params().BitSize
		coordSize := (curveBits + 7) / 8 // bytes per coordinate

		// Allocate the output buffer
		sig := make([]byte, 2*coordSize)

		// Copy R, left-padded
		sigR := ecdsaSig.SignatureR.Buffer
		if len(sigR) > coordSize {
			return nil, fmt.Errorf("TPM ECDSA signature R too long: got %d bytes, expected max %d", len(sigR), coordSize)
		}
		copy(sig[coordSize-len(sigR):coordSize], sigR)

		// Copy S, left-padded
		sigS := ecdsaSig.SignatureS.Buffer
		if len(sigS) > coordSize {
			return nil, fmt.Errorf("TPM ECDSA signature S too long: got %d bytes, expected max %d", len(sigS), coordSize)
		}
		copy(sig[2*coordSize-len(sigS):], sigS)

		// The final signature contains r||s, fixed-width, RFC 9421–compatible
		return sig, nil
	}

	// Standard ECDSA signature format for certificate signing requests
	// Convert TPM signature components to ASN.1 DER format
	sigR := new(big.Int).SetBytes(ecdsaSig.SignatureR.Buffer)
	sigS := new(big.Int).SetBytes(ecdsaSig.SignatureS.Buffer)

	// Encode as ASN.1 DER sequence manually
	type ecdsaSignature struct {
		R, S *big.Int
	}
	return asn1.Marshal(ecdsaSignature{R: sigR, S: sigS})
}
