//go:build linux

package tee

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpm2/transport/linuxtpm"
	"github.com/rs/zerolog"
)

const (
	// srkHandle is the standard handle for the Storage Root Key of TPM 2.0
	srkHandle = 0x81000001
)

// tpm2TEE implements the TEE interface using TPM 2.0.
type tpm2TEE struct {
	device          transport.TPMCloser
	logger          zerolog.Logger
	publicBlobPath  string
	privateBlobPath string
}

// TPM2Option is a functional option for configuring a TPM2 TEE
type TPM2Option func(*tpm2TEE)

// WithLogger sets the logger for the TPM2 TEE
func WithLogger(logger zerolog.Logger) TPM2Option {
	return func(t *tpm2TEE) {
		t.logger = logger
	}
}

// WithPublicBlobPath sets the path where the TPM public blob will be saved
func WithPublicBlobPath(path string) TPM2Option {
	return func(t *tpm2TEE) {
		t.publicBlobPath = path
	}
}

// WithPrivateBlobPath sets the path where the TPM private blob will be saved
func WithPrivateBlobPath(path string) TPM2Option {
	return func(t *tpm2TEE) {
		t.privateBlobPath = path
	}
}

// NewTPM2 creates a new TEE instance using TPM 2.0.
// It attempts to open the TPM device using the provided configuration.
func NewTPM2(opts ...TPM2Option) (TEE, error) {
	// Create TEE with default options
	t := &tpm2TEE{
		logger: zerolog.Nop(),
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	// Check that required options are set.
	if t.publicBlobPath == "" || t.privateBlobPath == "" {
		return nil, errors.New("required TPM2 options not set")
	}

	// Set up logger with component tag
	t.logger = t.logger.With().Str("component", "tee").Logger()
	t.logger.Info().Msg("Initializing TPM 2.0 connection")

	// Try opening the TPM 2.0 resource manager, which
	// - Provides managed access to TPM resources, allowing multiple applications to share the TPM safely.
	// - Used by the TPM2 Access Broker and Resource Manager (tpm2-abrmd or the kernel resource manager).
	devicePath := "/dev/tpmrm0"
	device, err := linuxtpm.Open(devicePath)
	if err != nil {
		// If TPM 2.0 resource manager is not available, fall back to the standard interface because
		// - Some systems (e.g. custom kernels, embedded devices, or misconfigured distros) don't expose /dev/tpmrm0.
		// - Better user experience — your app remains functional even in minimal environments.
		t.logger.Debug().Err(err).Msg("TPM resource manager not available, trying raw device")
		devicePath = "/dev/tpm0"
		device, err = linuxtpm.Open(devicePath)
		if err != nil {
			t.logger.Error().Err(err).Str("device_path", devicePath).Msg("Failed to open TPM device")
			return nil, ErrTEEUnavailable{Message: fmt.Sprintf("failed to open TPM device %s: %s", devicePath, err.Error())}
		}
	}
	t.logger.Info().Str("device_path", devicePath).Msg("Successfully opened TPM device")

	// Set the device
	t.device = device

	return t, nil
}

func (t *tpm2TEE) CreateKey(_ context.Context) (Key, error) {
	t.logger.Info().Msg("Creating new ECC key in TPM")

	parentKeyHandle, err := t.getOrCreateParentKey()
	if err != nil {
		return nil, fmt.Errorf("get or create TPM parent key: %w", err)
	}

	curveID, curveName := t.selectBestECCCurve()
	t.logger.Info().Str("curve", curveName).Msg("Selected ECC curve for key creation")

	// Create an ECC key template for the child key
	t.logger.Debug().Str("curve", curveName).Msg("Creating ECC key template")
	eccTemplate := tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			SignEncrypt:         true,
			Decrypt:             true,
		},
		Parameters: tpm2.NewTPMUPublicParms(
			tpm2.TPMAlgECC,
			&tpm2.TPMSECCParms{
				CurveID: curveID,
			},
		),
	})

	// Create the key under the persisted parent
	t.logger.Debug().Msg("Creating key under SRK")
	createKey, err := tpm2.Create{
		ParentHandle: parentKeyHandle,
		InPublic:     eccTemplate,
	}.Execute(t.device)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to create key under SRK")
		return nil, fmt.Errorf("create key under SRK: %w", err)
	}

	// Load the key
	t.logger.Debug().Msg("Loading created key")
	loadedKey, err := tpm2.Load{
		ParentHandle: parentKeyHandle,
		InPrivate:    createKey.OutPrivate,
		InPublic:     createKey.OutPublic,
	}.Execute(t.device)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to load key")
		return nil, fmt.Errorf("load key: %w", err)
	}

	t.logger.Debug().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("Key loaded successfully")

	// Save the key context
	t.logger.Debug().Msg("Saving key context")
	keyContext, err := tpm2.ContextSave{
		SaveHandle: loadedKey.ObjectHandle,
	}.Execute(t.device)
	cleanUp := func() {
		flush := tpm2.FlushContext{
			FlushHandle: loadedKey.ObjectHandle,
		}
		_, _ = flush.Execute(t.device)
	}
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to save key context")
		cleanUp()
		return nil, fmt.Errorf("save key context: %w", err)
	}

	t.logger.Info().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("Key created and context saved successfully")

	// Write TPM blobs to files
	if err := t.writeBlobsToFiles(createKey.OutPublic, createKey.OutPrivate); err != nil {
		t.logger.Error().Err(err).Msg("Failed to write TPM blobs to files")
		cleanUp()
		return nil, fmt.Errorf("write TPM blobs to files: %w", err)
	}

	// Create and return the key
	return &tpm2Key{
		tpm:         t.device,
		handle:      loadedKey.ObjectHandle,
		public:      createKey.OutPublic,
		context:     keyContext.Context,
		shouldFlush: true,
		logger:      t.logger,
	}, nil
}

// getOrCreateParentKey gets the Storage Root Key at the standard handle, or creates and persists it if it doesn't exist
func (t *tpm2TEE) getOrCreateParentKey() (tpm2.TPMHandle, error) {
	t.logger.Debug().Str("handle", fmt.Sprintf("0x%x", srkHandle)).Msg("Checking for existing SRK")

	// Try to read the public key at the SRK handle
	readPublic, err := tpm2.ReadPublic{
		ObjectHandle: tpm2.TPMHandle(srkHandle),
	}.Execute(t.device)
	if err == nil {
		// SRK exists, verify it has the required attributes
		t.logger.Debug().Msg("Found existing SRK, verifying attributes")

		pub, err := readPublic.OutPublic.Contents()
		if err != nil {
			return 0, fmt.Errorf("get SRK public contents: %w", err)
		}

		// Check required attributes that a persistent key must have.
		attrs := pub.ObjectAttributes
		if attrs.Decrypt && attrs.Restricted && attrs.FixedTPM &&
			attrs.FixedParent && attrs.SensitiveDataOrigin {
			t.logger.Info().Str("handle", fmt.Sprintf("0x%x", srkHandle)).Msg("Using existing SRK with correct attributes")
			return tpm2.TPMHandle(srkHandle), nil
		}

		// Otherwise fail because we do not handle a persistent key with wrong attributes at this point.
		err = fmt.Errorf("existing SRK has incorrect attributes: decrypt=%v, restricted=%v, fixedTPM=%v, fixedParent=%v, sensitiveDataOrigin=%v",
			attrs.Decrypt, attrs.Restricted, attrs.FixedTPM, attrs.FixedParent, attrs.SensitiveDataOrigin)
		t.logger.Err(err)
		return 0, err
	}

	// Create an SRK template with required attributes
	t.logger.Debug().Msg("Creating new RSA 2048-bit SRK")
	srkTemplate := tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgRSA,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
			SensitiveDataOrigin: true,
			Decrypt:             true,
			Restricted:          true,
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

	primaryKey, err := tpm2.CreatePrimary{
		PrimaryHandle: tpm2.TPMRHOwner,
		InPublic:      srkTemplate,
	}.Execute(t.device)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to create RSA SRK")
		return 0, fmt.Errorf("create RSA SRK: %w", err)
	}

	// Persist the SRK at the standard handle
	t.logger.Debug().Msg("Persisting SRK")
	_, err = tpm2.EvictControl{
		Auth:             tpm2.TPMRHOwner,
		ObjectHandle:     primaryKey.ObjectHandle,
		PersistentHandle: tpm2.TPMHandle(srkHandle),
	}.Execute(t.device)
	defer func() {
		// Clean up the transient object since the key has been persistent to a permanent handle.
		flush := tpm2.FlushContext{
			FlushHandle: primaryKey.ObjectHandle,
		}
		_, _ = flush.Execute(t.device)
	}()
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to persist SRK")
		return 0, fmt.Errorf("persist SRK: %w", err)
	}

	t.logger.Info().Str("handle", fmt.Sprintf("0x%x", srkHandle)).Msg("Created and persisted new SRK")
	return tpm2.TPMHandle(srkHandle), nil
}

// selectBestECCCurve checks if the TPM supports ECC P-384, otherwise returns P-256
func (t *tpm2TEE) selectBestECCCurve() (tpm2.TPMECCCurve, string) {
	t.logger.Debug().Msg("Checking TPM ECC curve support")

	// Try to create a test key with P-384 to check support
	// This is a more reliable method than querying capabilities
	testTemplate := tpm2.New2B(tpm2.TPMTPublic{
		Type:    tpm2.TPMAlgECC,
		NameAlg: tpm2.TPMAlgSHA256,
		ObjectAttributes: tpm2.TPMAObject{
			FixedTPM:            true,
			FixedParent:         true,
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

// writeBlobsToFiles writes the TPM public and private blobs to the specified file paths
func (t *tpm2TEE) writeBlobsToFiles(publicBlob tpm2.TPM2BPublic, privateBlob tpm2.TPM2BPrivate) error {
	t.logger.Debug().
		Str("public_path", t.publicBlobPath).
		Str("private_path", t.privateBlobPath).
		Msg("Writing TPM blobs to files")

	// Marshal the public blob
	publicData := tpm2.Marshal(publicBlob)
	if err := os.WriteFile(t.publicBlobPath, publicData, 0o600); err != nil {
		return fmt.Errorf("write public blob to %s: %w", t.publicBlobPath, err)
	}
	t.logger.Debug().
		Str("path", t.publicBlobPath).
		Int("size", len(publicData)).
		Msg("Public blob written successfully")

	// Marshal the private blob
	privateData := tpm2.Marshal(privateBlob)
	if err := os.WriteFile(t.privateBlobPath, privateData, 0o600); err != nil {
		return fmt.Errorf("write private blob to %s: %w", t.privateBlobPath, err)
	}
	t.logger.Debug().
		Str("path", t.privateBlobPath).
		Int("size", len(privateData)).
		Msg("Private blob written successfully")

	t.logger.Info().
		Str("public_path", t.publicBlobPath).
		Str("private_path", t.privateBlobPath).
		Msg("TPM blobs written to files successfully")

	return nil
}

func (t *tpm2TEE) LoadKey(_ context.Context) (Key, error) {
	t.logger.Info().
		Str("public_path", t.publicBlobPath).
		Str("private_path", t.privateBlobPath).
		Msg("Loading key from TPM blobs")

	// Read public blob from file
	t.logger.Debug().Str("path", t.publicBlobPath).Msg("Reading public blob")
	publicData, err := os.ReadFile(t.publicBlobPath)
	if err != nil {
		t.logger.Error().Err(err).Str("path", t.publicBlobPath).Msg("Failed to read public blob")
		return nil, fmt.Errorf("read public blob from %s: %w", t.publicBlobPath, err)
	}

	// Read private blob from file
	t.logger.Debug().Str("path", t.privateBlobPath).Msg("Reading private blob")
	privateData, err := os.ReadFile(t.privateBlobPath)
	if err != nil {
		t.logger.Error().Err(err).Str("path", t.privateBlobPath).Msg("Failed to read private blob")
		return nil, fmt.Errorf("read private blob from %s: %w", t.privateBlobPath, err)
	}

	// Unmarshal the blobs
	t.logger.Debug().Msg("Unmarshaling TPM blobs")
	publicBlob, err := tpm2.Unmarshal[tpm2.TPM2BPublic](publicData)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to unmarshal public blob")
		return nil, fmt.Errorf("unmarshal public blob: %w", err)
	}

	privateBlob, err := tpm2.Unmarshal[tpm2.TPM2BPrivate](privateData)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to unmarshal private blob")
		return nil, fmt.Errorf("unmarshal private blob: %w", err)
	}

	// Get the parent key handle (SRK)
	parentKeyHandle, err := t.getOrCreateParentKey()
	if err != nil {
		return nil, fmt.Errorf("get parent key: %w", err)
	}

	// Load the key using the parent SRK
	t.logger.Debug().Uint32("parent_handle", uint32(parentKeyHandle)).Msg("Loading key under SRK")
	loadedKey, err := tpm2.Load{
		ParentHandle: parentKeyHandle,
		InPrivate:    *privateBlob,
		InPublic:     *publicBlob,
	}.Execute(t.device)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to load key under SRK")
		return nil, fmt.Errorf("load key under SRK: %w", err)
	}

	t.logger.Debug().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("Key loaded successfully")

	// Save the key context for potential future use
	t.logger.Debug().Msg("Saving key context")
	keyContext, err := tpm2.ContextSave{
		SaveHandle: loadedKey.ObjectHandle,
	}.Execute(t.device)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to save key context")
		// Clean up
		flush := tpm2.FlushContext{
			FlushHandle: loadedKey.ObjectHandle,
		}
		_, _ = flush.Execute(t.device)
		return nil, fmt.Errorf("save key context: %w", err)
	}

	t.logger.Info().
		Str("handle", fmt.Sprintf("0x%x", loadedKey.ObjectHandle)).
		Msg("Key loaded from blobs successfully")

	return &tpm2Key{
		tpm:         t.device,
		handle:      loadedKey.ObjectHandle,
		public:      *publicBlob,
		context:     keyContext.Context,
		shouldFlush: true,
		logger:      t.logger,
	}, nil
}

func (t *tpm2TEE) Close() error {
	t.logger.Info().Msg("Closing TPM device")
	if t.device != nil {
		err := t.device.Close()
		if err != nil {
			t.logger.Error().Err(err).Msg("Error closing TPM device")
			return err
		}
		t.logger.Debug().Msg("TPM device closed successfully")
	}
	return nil
}

// tpm2Key implements the Key interface using TPM 2.0.
type tpm2Key struct {
	tpm         transport.TPMCloser
	handle      tpm2.TPMHandle
	public      tpm2.TPM2BPublic
	context     tpm2.TPMSContext
	shouldFlush bool
	logger      zerolog.Logger
}

func (k *tpm2Key) Signer() (crypto.Signer, error) {
	// Parse public key
	pub, err := k.public.Contents()
	if err != nil {
		return nil, fmt.Errorf("get public key contents: %w", err)
	}

	if pub.Type != tpm2.TPMAlgECC {
		return nil, fmt.Errorf("not an ECC key")
	}

	eccDetail, err := pub.Parameters.ECCDetail()
	if err != nil {
		return nil, fmt.Errorf("get ECC details: %w", err)
	}

	eccUnique, err := pub.Unique.ECC()
	if err != nil {
		return nil, fmt.Errorf("get ECC unique: %w", err)
	}

	// Create crypto.PublicKey based on curve
	var publicKey crypto.PublicKey
	switch eccDetail.CurveID {
	case tpm2.TPMECCNistP256:
		publicKey = &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(eccUnique.X.Buffer),
			Y:     new(big.Int).SetBytes(eccUnique.Y.Buffer),
		}
	case tpm2.TPMECCNistP384:
		publicKey = &ecdsa.PublicKey{
			Curve: elliptic.P384(),
			X:     new(big.Int).SetBytes(eccUnique.X.Buffer),
			Y:     new(big.Int).SetBytes(eccUnique.Y.Buffer),
		}
	default:
		return nil, fmt.Errorf("unsupported ECC curve: %v", eccDetail.CurveID)
	}

	return &tpm2Signer{
		tpm:       k.tpm,
		handle:    k.handle,
		publicKey: publicKey,
	}, nil
}

func (k *tpm2Key) Public() (crypto.PublicKey, error) {
	signer, err := k.Signer()
	if err != nil {
		return nil, err
	}
	return signer.Public(), nil
}

func (k *tpm2Key) Marshal() ([]byte, error) {
	return tpm2.Marshal(k.context), nil
}

func (k *tpm2Key) Close() error {
	if k.shouldFlush && k.handle != 0 {
		flush := tpm2.FlushContext{
			FlushHandle: k.handle,
		}
		_, err := flush.Execute(k.tpm)
		k.handle = 0
		return err
	}
	return nil
}

// tpm2Signer implements crypto.Signer using TPM 2.0.
type tpm2Signer struct {
	tpm       transport.TPMCloser
	handle    tpm2.TPMHandle
	publicKey crypto.PublicKey
}

func (s *tpm2Signer) Public() crypto.PublicKey {
	return s.publicKey
}

func (s *tpm2Signer) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// Determine hash algorithm
	hashAlg := tpm2.TPMAlgSHA256
	switch opts.HashFunc() {
	case crypto.SHA1:
		hashAlg = tpm2.TPMAlgSHA1
	case crypto.SHA256:
		hashAlg = tpm2.TPMAlgSHA256
	case crypto.SHA384:
		hashAlg = tpm2.TPMAlgSHA384
	case crypto.SHA512:
		hashAlg = tpm2.TPMAlgSHA512
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

	// Convert TPM ECDSA signature to standard format
	// TPM returns r and s as separate values, we need to encode them properly
	rBytes := ecdsaSig.SignatureR.Buffer
	sBytes := ecdsaSig.SignatureS.Buffer

	// Create ASN.1 DER encoded signature
	return encodeECDSASignature(rBytes, sBytes)
}

// ecdsaSignature represents an ECDSA signature in ASN.1 DER format
type ecdsaSignature struct {
	R, S *big.Int
}

// encodeECDSASignature encodes r and s values into ASN.1 DER format
func encodeECDSASignature(rBytes, sBytes []byte) ([]byte, error) {
	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)

	sig := ecdsaSignature{R: r, S: s}
	return asn1.Marshal(sig)
}
