package acme

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	httpmdm "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/mdm"

	stepconfig "github.com/smallstep/certificates/authority/config"
	stepca "github.com/smallstep/certificates/ca"
	"go.step.sm/crypto/keyutil"
	"go.step.sm/crypto/pemutil"
	"go.step.sm/crypto/x509util"
)

type Service interface {
	Stop() error
	Verify(ctx context.Context, cert *x509.Certificate) error
}

type acmeService struct {
	stepca       *stepca.CA
	root         *x509.Certificate
	intermediate *x509.Certificate
	signer       crypto.Signer
}

// compile-time assertion that acmeService implements the Service and httpmdm.CertVerifier interfaces.
var (
	_ Service              = (*acmeService)(nil)
	_ httpmdm.CertVerifier = (*acmeService)(nil)
)

func (a *acmeService) Verify(ctx context.Context, cert *x509.Certificate) error {
	opts := x509.VerifyOptions{
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		Roots:         x509.NewCertPool(),
		Intermediates: x509.NewCertPool(),
	}
	opts.Roots.AddCert(a.root)
	opts.Intermediates.AddCert(a.intermediate)
	if _, err := cert.Verify(opts); err != nil {
		return ctxerr.Wrap(ctx, err, "acmer verifier")
	}
	return nil
}

func (a *acmeService) Stop() error {
	if a.stepca == nil {
		return nil
	}
	return a.stepca.Stop()
}

func StartNewService(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) (Service, error) {
	root, intermediate, signer, err := setupAssets(ctx, ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "start acme: setup assets")
	}

	ca, err := newStepCA(root, intermediate, signer)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "start acme: new step CA")
	}

	// TODO: There's plenty more we'll want to do to make this production ready, but for this PoC
	// we'll just spin out a go routine to run an instance of step-ca with an ACME provisioner.
	// Most likley, we want to orchestrate this differently, but that requires some deeper thinking about
	// our architecture and some more unpacking of the step-ca codebase. As currently implemented,
	// step-ca listens on its own port and does it own routing. For the PoC it is creating its own MySQL
	// database, but it could be configured to use the main Fleet DB if we want to add the step-ca
	// schema.
	go func() {
		if err := ca.Run(); err != nil {
			level.Error(logger).Log("msg", "step-ca error", "err", err)
			ctxerr.Handle(ctx, err)
		}
	}()

	return &acmeService{stepca: ca, root: root, intermediate: intermediate, signer: signer}, nil
}

// TODO: we'll likely need to persist the intermediate certificate and signer in order to be able to
// verify certs issued by the ACME provisioner across restarts.
func setupAssets(ctx context.Context, ds fleet.Datastore) (*x509.Certificate, *x509.Certificate, crypto.Signer, error) {
	kp, err := assets.CAKeyPair(ctx, ds)
	if err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "loading CA key pair")
	}
	signer, err := keyutil.GenerateDefaultSigner()
	if err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "generating CA signer")
	}

	intCR, err := x509util.CreateCertificateRequest("Fleet ACME Intermediate CA", []string{}, signer)
	if err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "create acme intermediate cert request")
	}

	cert, err := x509util.NewCertificate(intCR, x509util.WithTemplate(x509util.DefaultIntermediateTemplate, x509util.CreateTemplateData("Fleet ACME Intermediate CA", []string{})))
	if err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "create new acme intermediate cert")
	}

	template := cert.GetCertificate()
	template.NotBefore = time.Now()
	template.NotAfter = time.Now().Add(24 * time.Hour)

	intermediate, err := x509util.CreateCertificate(template, kp.Leaf, signer.Public(), kp.PrivateKey.(crypto.Signer))
	if err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "create acme intermediate cert")
	}

	return kp.Leaf, intermediate, signer, nil
}

func newStepCA(root *x509.Certificate, intermediate *x509.Certificate, signer crypto.Signer) (*stepca.CA, error) {
	// read step-ca config from file path specified in FLEET_DEV_STEP_CA_CONFIG environment variable
	stepFilepath := os.Getenv("FLEET_DEV_STEP_CA_CONFIG")
	if stepFilepath == "" {
		return nil, fmt.Errorf("FLEET_DEV_STEP_CA_CONFIG environment variable not set")
	}
	var stepCfg stepconfig.Config
	b, err := os.ReadFile(stepFilepath)
	if err != nil {
		return nil, fmt.Errorf("reading step-ca config from file: %w", err)
	}
	if err := json.Unmarshal(b, &stepCfg); err != nil {
		return nil, fmt.Errorf("unmarshalling step-ca config: %w", err)
	}

	// TODO: there is certainly a better way to do all this, but for now we'll write the certs and key
	// to disk and point step-ca to those files because step-ca expects file paths for the root and
	// intermediate certs and keys
	tmpPath := os.TempDir()
	writeCert := func(fn string, certs ...*x509.Certificate) error {
		var b []byte
		for _, crt := range certs {
			b = append(b, pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: crt.Raw,
			})...)
		}
		return os.WriteFile(filepath.Join(tmpPath, fn), b, 0o600)
	}
	writeKey := func(fn string, signer crypto.Signer) error {
		_, err := pemutil.Serialize(signer, pemutil.ToFile(filepath.Join(tmpPath, fn), 0o600))
		return err
	}
	if err := writeCert("root0.crt", root); err != nil {
		return nil, err
	}
	if err := writeCert("int0.crt", intermediate); err != nil {
		return nil, err
	}
	if err := writeKey("int0.key", signer); err != nil {
		return nil, err
	}
	stepCfg.Root = []string{filepath.Join(tmpPath, "root0.crt")}
	stepCfg.IntermediateCert = filepath.Join(tmpPath, "int0.crt")
	stepCfg.IntermediateKey = filepath.Join(tmpPath, "int0.key")

	return stepca.New(&stepCfg)
}

// // minica is a handy utility that I found from smallstep to create a simple CA for testing purposes.
// func setupMiniCA() (*minica.CA, error) {
// 	mca, err := minica.New()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return mca, nil
// }

// // stepRouter is a experimental adapter to map step-ca routes (which use go-chi) to our existing
// // patterns (which use gorilla/mux). It worked OK in testing as far as routing, but there's quite a
// // bit of middleware and context-based dependency-iinjection in step-ca that would need to be
// // replicated in order to refine this approach.
// type stepRouter struct {
// 	router *mux.Router
// 	logger kitlog.Logger
// }

// // MethodFunc implements the smallstep api.Router interface and registers the given
// // handler for the given method and path to an underlying gorilla/mux Router.
// func (sr *stepRouter) MethodFunc(method, path string, fn http.HandlerFunc) {
// 	sr.router.Path(path).Methods(method).HandlerFunc(fn)
// }

// func newStepRouter(logger kitlog.Logger) *stepRouter {
// 	return &stepRouter{router: mux.NewRouter(), logger: logger}
// }
