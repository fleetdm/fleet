package apple_mdm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
)

// pkcs12CAType is a kind of certificate authority that gets each host its own PKCS#12 certificate, through a pair of profile variables
// for the certificate and its password. A host's certificate is requested when its profile is processed.
type pkcs12CAType struct {
	vendor                     string
	dataPrefix, passwordPrefix fleet.FleetVarName
	managedType                fleet.CAConfigAssetType
	// varHint is added to the failure detail of a variable of the CA, to tell the admin it is not in the profile.
	varHint string
	// find returns the named CA of the type, which has its settings and the service that issues its certificates.
	find func(groupedCAs *fleet.GroupedCertificateAuthorities, name string) (pkcs12CA, bool)
}

// pkcs12CA is one configured certificate authority.
type pkcs12CA interface {
	// forHost returns the fields of a copy of the CA that have Fleet variables to fill in for a host, and the function that requests the
	// certificate with those fields once they are filled in.
	forHost() (fields []*string, issue func(ctx context.Context) (*fleet.IssuedCertificate, error))
}

type digiCertCA struct {
	ca      fleet.DigiCertCA
	service fleet.DigiCertService
}

func (d digiCertCA) forHost() ([]*string, func(context.Context) (*fleet.IssuedCertificate, error)) {
	ca := d.ca
	// Deep copy the UPN slice to prevent cross-host contamination: a shallow copy shares the backing array, so in-place substitutions for
	// one host would corrupt the cached CA used by subsequent hosts.
	ca.CertificateUserPrincipalNames = slices.Clone(d.ca.CertificateUserPrincipalNames)
	fields := []*string{&ca.CertificateCommonName, &ca.CertificateSeatID}
	for i := range ca.CertificateUserPrincipalNames {
		fields = append(fields, &ca.CertificateUserPrincipalNames[i])
	}
	return fields, func(ctx context.Context) (*fleet.IssuedCertificate, error) { return d.service.GetCertificate(ctx, ca) }
}

type ejbcaCA struct {
	ca      fleet.EJBCA
	service fleet.EJBCAService
}

func (e ejbcaCA) forHost() ([]*string, func(context.Context) (*fleet.IssuedCertificate, error)) {
	ca := e.ca
	ca.CertificateUserPrincipalNames = slices.Clone(e.ca.CertificateUserPrincipalNames)
	if ca.UsernameTemplate == "" {
		ca.UsernameTemplate = ca.CertificateCommonName
	}
	fields := []*string{&ca.CertificateCommonName, &ca.UsernameTemplate}
	for i := range ca.CertificateUserPrincipalNames {
		fields = append(fields, &ca.CertificateUserPrincipalNames[i])
	}
	return fields, func(ctx context.Context) (*fleet.IssuedCertificate, error) { return e.service.GetCertificate(ctx, ca) }
}

func newPKCS12CATypes(digiCertService fleet.DigiCertService, ejbcaService fleet.EJBCAService) []pkcs12CAType {
	return []pkcs12CAType{
		{
			vendor:         "DigiCert",
			dataPrefix:     fleet.FleetVarDigiCertDataPrefix,
			passwordPrefix: fleet.FleetVarDigiCertPasswordPrefix,
			managedType:    fleet.CAConfigDigiCert,
			varHint:        digiCertVarHint,
			find: func(groupedCAs *fleet.GroupedCertificateAuthorities, name string) (pkcs12CA, bool) {
				for _, ca := range groupedCAs.DigiCert {
					if ca.Name == name {
						return digiCertCA{ca: ca, service: digiCertService}, true
					}
				}
				return nil, false
			},
		},
		{
			vendor:         "EJBCA",
			dataPrefix:     fleet.FleetVarEJBCADataPrefix,
			passwordPrefix: fleet.FleetVarEJBCAPasswordPrefix,
			managedType:    fleet.CAConfigEJBCA,
			varHint:        ejbcaVarHint,
			find: func(groupedCAs *fleet.GroupedCertificateAuthorities, name string) (pkcs12CA, bool) {
				for _, ca := range groupedCAs.EJBCA {
					if ca.Name == name {
						return ejbcaCA{ca: ca, service: ejbcaService}, true
					}
				}
				return nil, false
			},
		},
	}
}

// pkcs12CAVariable is a Fleet variable of a PKCS#12 certificate authority found in a profile.
type pkcs12CAVariable struct {
	caType *pkcs12CAType
	caName string
	// isData is for the variable of the certificate, as opposed to the one of its password.
	isData bool
}

// findPKCS12CAVariable returns what the Fleet variable (without the FLEET_VAR_ prefix) is, if it is one of a PKCS#12 certificate authority.
func findPKCS12CAVariable(caTypes []pkcs12CAType, fleetVar string) (pkcs12CAVariable, bool) {
	for i := range caTypes {
		if name, ok := strings.CutPrefix(fleetVar, string(caTypes[i].dataPrefix)); ok {
			return pkcs12CAVariable{caType: &caTypes[i], caName: name, isData: true}, true
		}
		if name, ok := strings.CutPrefix(fleetVar, string(caTypes[i].passwordPrefix)); ok {
			return pkcs12CAVariable{caType: &caTypes[i], caName: name}, true
		}
	}
	return pkcs12CAVariable{}, false
}

// cacheKey is the key of the CA in the cache of a profile's CAs.
func (v pkcs12CAVariable) cacheKey() string {
	return string(v.caType.dataPrefix) + v.caName
}

// isPKCS12CAConfigured checks that the CA of the variable exists, and caches it. If not, it marks the profile failed for the hosts of the target.
func isPKCS12CAConfigured(ctx context.Context, logger *slog.Logger, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string, cache map[string]pkcs12CA, profUUID string, target *fleet.CmdTarget,
	variable pkcs12CAVariable, fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			variable.caType.vendor+" integration requires a Fleet Premium license.", new(time.Now().UTC()))
	}
	if _, ok := cache[variable.cacheKey()]; ok {
		return true, nil
	}
	if groupedCAs != nil {
		if ca, ok := variable.caType.find(groupedCAs, variable.caName); ok {
			cache[variable.cacheKey()] = ca
			return true, nil
		}
	}
	return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
		fmt.Sprintf("Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, variable.caName), new(time.Now().UTC()))
}

// issuePKCS12ForHost fills in the CA's fields for the host, requests its certificate, and puts the certificate and its password in the
// profile contents. If something fails, the host's profile is marked failed and ok is false, except when the CA is temporarily
// unavailable: then the profile is retried later, as many times as a profile that fails on the host, and only marked failed after that. The
// managed certificate is only for the caller to record once the whole profile is sent.
func issuePKCS12ForHost(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, ca pkcs12CA, variable pkcs12CAVariable, target *fleet.CmdTarget,
	hostLite fleet.Host, profUUID, hostContents string, hostIDForUUIDCache map[string]uint, variablesUpdatedAt *time.Time,
	onMismatchedHostCount func(int) error,
) (contents string, managed *fleet.MDMManagedCertificate, ok bool, err error) {
	fields, issue := ca.forHost()
	// The fields share the values of the host that are looked up once.
	caVarsCache := make(map[string]string)
	for _, field := range fields {
		ok, err := replaceFleetVarInCAItem(ctx, ds, target, hostLite, caVarsCache, field, hostIDForUUIDCache, variablesUpdatedAt,
			onMismatchedHostCount, variable.caType.varHint)
		if err != nil {
			return "", nil, false, ctxerr.Wrapf(ctx, err, "populating Fleet variables in %s CA fields", variable.caType.vendor)
		}
		if !ok {
			return "", nil, false, nil
		}
	}

	cert, err := issue(ctx)
	if err != nil {
		if _, transient := errors.AsType[fleet.CertificateAuthorityTransientError](err); transient {
			retrying, retryErr := retryHostProfile(ctx, ds, hostLite.UUID, target.ProfileIdentifier)
			if retryErr != nil {
				return "", nil, false, ctxerr.Wrapf(ctx, retryErr, "retrying host MDM Apple profile for %s", variable.caType.vendor)
			}
			if retrying {
				logger.WarnContext(ctx, "could not get a certificate because the CA is unavailable; the profile will be retried",
					"vendor", variable.caType.vendor, "ca", variable.caName, "host_uuid", hostLite.UUID, "profile_uuid", profUUID, "err", err)
				return "", nil, false, nil
			}
		}
		err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			CommandUUID:        target.CmdUUID,
			HostUUID:           hostLite.UUID,
			Status:             &fleet.MDMDeliveryFailed,
			Detail:             fmt.Sprintf("Couldn't get certificate from %s for %s. %s", variable.caType.vendor, variable.caName, err),
			OperationType:      fleet.MDMOperationTypeInstall,
			VariablesUpdatedAt: variablesUpdatedAt,
		})
		if err != nil {
			return "", nil, false, ctxerr.Wrapf(ctx, err, "updating host MDM Apple profile for %s", variable.caType.vendor)
		}
		return "", nil, false, nil
	}
	hostContents, err = profiles.ReplaceExactFleetPrefixVariableInXML(string(variable.caType.dataPrefix), variable.caName, hostContents,
		base64.StdEncoding.EncodeToString(cert.PfxData))
	if err != nil {
		return "", nil, false, ctxerr.Wrapf(ctx, err, "replacing Fleet variable for %s data", variable.caType.vendor)
	}
	hostContents, err = profiles.ReplaceExactFleetPrefixVariableInXML(string(variable.caType.passwordPrefix), variable.caName, hostContents,
		cert.Password)
	if err != nil {
		return "", nil, false, ctxerr.Wrapf(ctx, err, "replacing Fleet variable for %s password", variable.caType.vendor)
	}
	return hostContents, &fleet.MDMManagedCertificate{
		HostUUID:       hostLite.UUID,
		ProfileUUID:    profUUID,
		NotValidBefore: &cert.NotValidBefore,
		NotValidAfter:  &cert.NotValidAfter,
		Type:           variable.caType.managedType,
		CAName:         variable.caName,
		Serial:         &cert.SerialNumber,
	}, true, nil
}

// retryHostProfile queues the host's profile to be processed again, if it has retries left, and reports whether it did. It uses the same
// count as a profile that the host fails to install, so the profile is only marked failed once that is used up.
func retryHostProfile(ctx context.Context, ds fleet.Datastore, hostUUID, profileIdentifier string) (bool, error) {
	// The platform is darwin for iOS and iPadOS too, which have the same profiles and retries.
	host := &fleet.Host{UUID: hostUUID, Platform: "darwin"}
	counts, err := ds.GetHostMDMProfilesRetryCounts(ctx, host)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "getting profile retry counts")
	}
	for _, count := range counts {
		if count.ProfileIdentifier == profileIdentifier && count.Retries >= mdm.MaxAppleProfileRetries {
			return false, nil
		}
	}
	if err := ds.UpdateHostMDMProfilesVerification(ctx, host, nil, nil, []string{profileIdentifier}); err != nil {
		return false, ctxerr.Wrap(ctx, err, "queueing profile retry")
	}
	return true, nil
}
