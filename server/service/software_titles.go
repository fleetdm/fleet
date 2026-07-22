package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

/////////////////////////////////////////////////////////////////////////////////
// List Software Titles
/////////////////////////////////////////////////////////////////////////////////

type listSoftwareTitlesRequest struct {
	fleet.SoftwareTitleListOptions
}

type listSoftwareTitlesResponse struct {
	Meta            *fleet.PaginationMetadata       `json:"meta"`
	Count           int                             `json:"count"`
	CountsUpdatedAt *time.Time                      `json:"counts_updated_at"`
	SoftwareTitles  []fleet.SoftwareTitleListResult `json:"software_titles"`
	Err             error                           `json:"error,omitempty"`
}

func (r listSoftwareTitlesResponse) Error() error { return r.Err }

func listSoftwareTitlesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listSoftwareTitlesRequest)
	titles, count, meta, err := svc.ListSoftwareTitles(ctx, req.SoftwareTitleListOptions)
	if err != nil {
		return listSoftwareTitlesResponse{Err: err}, nil
	}

	var latest time.Time
	for _, sw := range titles {
		if sw.CountsUpdatedAt != nil && !sw.CountsUpdatedAt.IsZero() && sw.CountsUpdatedAt.After(latest) {
			latest = *sw.CountsUpdatedAt
		}
		// we dont want to include the InstallDuringSetup field in the response
		// for software titles list.
		if sw.SoftwarePackage != nil {
			sw.SoftwarePackage.InstallDuringSetup = nil
		} else if sw.AppStoreApp != nil {
			sw.AppStoreApp.InstallDuringSetup = nil
		}
	}
	if len(titles) == 0 {
		titles = []fleet.SoftwareTitleListResult{}
	}
	listResp := listSoftwareTitlesResponse{
		SoftwareTitles: titles,
		Count:          count,
		Meta:           meta,
	}
	if !latest.IsZero() {
		listResp.CountsUpdatedAt = &latest
	}

	return listResp, nil
}

func (svc *Service) ListSoftwareTitles(
	ctx context.Context,
	opt fleet.SoftwareTitleListOptions,
) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, 0, nil, err
	}

	lic, err := svc.License(ctx)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "get license")
	}

	if opt.TeamID != nil && *opt.TeamID != 0 && !lic.IsPremium() {
		return nil, 0, nil, fleet.ErrMissingLicense
	}

	if !lic.IsPremium() && (opt.MaximumCVSS > 0 || opt.MinimumCVSS > 0 || opt.KnownExploit) {
		return nil, 0, nil, fleet.ErrMissingLicense
	}

	// always include metadata for software titles
	opt.ListOptions.IncludeMetadata = true
	// cursor-based pagination is not supported for software titles
	opt.ListOptions.After = ""

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, 0, nil, fleet.ErrNoContext
	}

	titles, count, meta, err := svc.ds.ListSoftwareTitles(ctx, opt, fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
		TeamID:          opt.TeamID,
	})
	if err != nil {
		return nil, 0, nil, err
	}

	return titles, count, meta, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Get a Software Title
/////////////////////////////////////////////////////////////////////////////////

type getSoftwareTitleRequest struct {
	ID     uint  `url:"id"`
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type getSoftwareTitleResponse struct {
	SoftwareTitle *fleet.SoftwareTitle `json:"software_title,omitempty"`
	Err           error                `json:"error,omitempty"`
}

func (r getSoftwareTitleResponse) Error() error { return r.Err }

func getSoftwareTitleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getSoftwareTitleRequest)

	software, err := svc.SoftwareTitleByID(ctx, req.ID, req.TeamID)
	if err != nil {
		return getSoftwareTitleResponse{Err: err}, nil
	}

	return getSoftwareTitleResponse{SoftwareTitle: software}, nil
}

func (svc *Service) SoftwareTitleByID(ctx context.Context, id uint, teamID *uint) (*fleet.SoftwareTitle, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	if teamID != nil {
		// Verify the caller has permission for the requested scope (team or global).
		if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{TeamID: teamID}, fleet.ActionRead); err != nil {
			return nil, err
		}
		if *teamID != 0 {
			exists, err := svc.ds.TeamExists(ctx, *teamID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "checking if team exists")
			} else if !exists {
				return nil, fleet.NewInvalidArgumentError("team_id/fleet_id", fmt.Sprintf("fleet %d does not exist", *teamID)).
					WithStatus(http.StatusNotFound)
			}
		}
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	// get software by id including team_id data from software_title_host_counts
	software, err := svc.ds.SoftwareTitleByID(ctx, id, teamID, fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
	})
	if err != nil {
		if fleet.IsNotFound(err) && teamID == nil {
			// here we use a global admin as filter because we want to check if the software exists
			filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
			_, err = svc.ds.SoftwareTitleByID(ctx, id, nil, filter)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "checked using a global admin")
			}

			return nil, fleet.NewPermissionError("Error: You don't have permission to view specified software. It is installed on hosts that belong to a fleet you don't have permissions to view.")
		}
		return nil, ctxerr.Wrap(ctx, err, "getting software title by id")
	}

	license, err := svc.License(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get license")
	}
	if license.IsPremium() {
		// add software installer data if needed
		if software.SoftwareInstallersCount > 0 {
			pkgs, err := svc.ds.GetSoftwarePackagesByTeamAndTitleID(ctx, teamID, id)
			if err != nil && !fleet.IsNotFound(err) {
				return nil, ctxerr.Wrap(ctx, err, "get software packages")
			}
			if len(pkgs) > 0 {
				// Display name and icon are title-level; fetch once from the first-added package.
				titleMeta, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, id, true)
				if err != nil && !fleet.IsNotFound(err) {
					return nil, ctxerr.Wrap(ctx, err, "get software installer metadata")
				}

				// Categories are per-package.
				installerIDs := make([]uint, len(pkgs))
				for i, pkg := range pkgs {
					installerIDs[i] = pkg.InstallerID
				}
				categoriesByInstaller, err := svc.ds.GetCategoriesForSoftwareInstallers(ctx, installerIDs)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "get categories for software packages")
				}

				// Key policies by installer_id so each package on a multi-package
				// title only surfaces the ones actually bound to it. VPP-backed
				// policies have nil InstallerID and dispatch via AppStoreApp.
				policiesByInstaller := make(map[uint][]fleet.AutomaticInstallPolicy)
				if titleMeta != nil {
					for _, p := range titleMeta.AutomaticInstallPolicies {
						if p.InstallerID == nil {
							continue
						}
						policiesByInstaller[*p.InstallerID] = append(policiesByInstaller[*p.InstallerID], p)
					}
				}

				for _, pkg := range pkgs {
					summary, err := svc.ds.GetSummaryHostSoftwareInstalls(ctx, pkg.InstallerID)
					if err != nil {
						return nil, ctxerr.Wrap(ctx, err, "get software installer status summary")
					}
					pkg.Status = summary
					pkg.Categories = categoriesByInstaller[pkg.InstallerID]

					if titleMeta != nil {
						pkg.DisplayName = titleMeta.DisplayName
						pkg.IconUrl = titleMeta.IconUrl
					}
					pkg.AutomaticInstallPolicies = policiesByInstaller[pkg.InstallerID]

					// Populate FleetMaintainedVersions/pin/patch policy for FMA titles.
					// An FMA title has a single active package, so this runs on it.
					if pkg.FleetMaintainedAppID != nil {
						fmaVersions, err := svc.ds.GetFleetMaintainedVersionsByTitleID(ctx, teamID, id, false)
						if err != nil {
							return nil, ctxerr.Wrap(ctx, err, "get fleet maintained versions")
						}
						pkg.FleetMaintainedVersions = fmaVersions

						// No pin row means the title tracks "Latest" (nil pinned_version); any other error is real.
						pinnedVersion, err := svc.ds.GetPinnedVersion(ctx, teamID, id)
						if err != nil && !errors.Is(err, sql.ErrNoRows) {
							return nil, ctxerr.Wrap(ctx, err, "get pinned version")
						}
						pkg.PinnedVersion = pinnedVersion

						patchPolicy, err := svc.ds.GetPatchPolicy(ctx, teamID, id)
						if err != nil && !fleet.IsNotFound(err) {
							return nil, ctxerr.Wrap(ctx, err, "get patch policy")
						}
						pkg.PatchPolicy = patchPolicy
					}
				}

				// software_package is kept for backwards compatibility and equals the first-added package.
				software.Packages = make([]fleet.SoftwareInstaller, len(pkgs))
				for i, pkg := range pkgs {
					software.Packages[i] = *pkg
				}
				software.SoftwarePackage = pkgs[0]
			}
		}

		// add VPP app data if needed
		if software.VPPAppsCount > 0 {
			meta, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, teamID, id)
			if err != nil && !fleet.IsNotFound(err) {
				return nil, ctxerr.Wrap(ctx, err, "get VPP app metadata")
			}
			if meta != nil {
				summary, err := svc.ds.GetSummaryHostVPPAppInstalls(ctx, teamID, meta.VPPAppID)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "get VPP app status summary")
				}
				meta.Status = summary

				// Wrap iOS / iPadOS plist as a JSON string for the response.
				if len(meta.Configuration) > 0 {
					switch meta.Platform {
					case fleet.IOSPlatform, fleet.IPadOSPlatform:
						wrapped, err := json.Marshal(string(meta.Configuration))
						if err != nil {
							return nil, ctxerr.Wrap(ctx, err, "wrapping VPP configuration for response")
						}
						meta.Configuration = wrapped
					}
				}
			}
			software.AppStoreApp = meta
		}

		// add in house app data if needed
		if software.InHouseAppCount > 0 {
			meta, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, teamID, id)
			if err != nil && !fleet.IsNotFound(err) {
				return nil, ctxerr.Wrap(ctx, err, "get in house app metadata")
			}
			if meta != nil {
				summary, err := svc.ds.GetSummaryHostInHouseAppInstalls(ctx, teamID, meta.InstallerID)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "get in house app status summary")
				}
				meta.Status = &fleet.SoftwareInstallerStatusSummary{
					Installed:      summary.Installed,
					PendingInstall: summary.Pending,
					FailedInstall:  summary.Failed,
				}

				// Wrap iOS / iPadOS plist as a JSON string for the response.
				if len(meta.Configuration) > 0 {
					wrapped, err := json.Marshal(string(meta.Configuration))
					if err != nil {
						return nil, ctxerr.Wrap(ctx, err, "wrapping in-house app configuration for response")
					}
					meta.Configuration = wrapped
				}
			}
			software.SoftwarePackage = meta
		}
	}

	return software, nil
}

func (svc *Service) SoftwareTitleNameForHostFilter(ctx context.Context, id uint, teamID *uint) (name, displayName string, err error) {
	// Intentionally skip team-scoped inventory auth: only minimal title name.
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return "", "", err
	}

	name, displayName, err = svc.ds.SoftwareTitleNameForHostFilter(ctx, id, teamID)
	if err != nil {
		return "", "", err
	}

	return name, displayName, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Update a software title's name
/////////////////////////////////////////////////////////////////////////////////

type updateSoftwareNameRequest struct {
	ID   uint   `url:"id"`
	Name string `json:"name"`
}

type updateSoftwareNameResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateSoftwareNameResponse) Error() error { return r.Err }
func (r updateSoftwareNameResponse) Status() int  { return http.StatusResetContent }

func updateSoftwareNameEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*updateSoftwareNameRequest)
	return updateSoftwareNameResponse{Err: svc.UpdateSoftwareName(ctx, req.ID, req.Name)}, nil
}

func (svc *Service) UpdateSoftwareName(ctx context.Context, titleID uint, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{}, fleet.ActionWrite); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	// get software by id including team_id data from software_title_host_counts
	software, err := svc.ds.SoftwareTitleByID(ctx, titleID, nil, fleet.TeamFilter{
		User: vc.User,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting software title by id")
	}
	if software.BundleIdentifier == nil || *software.BundleIdentifier == "" {
		return fleet.NewInvalidArgumentError("id", "only titles with a bundle ID can have their name modified")
	}
	if name == "" {
		return fleet.NewInvalidArgumentError("name", "cannot be empty")
	}

	return svc.ds.UpdateSoftwareTitleName(ctx, titleID, name)
}

func (svc *Service) UpdateSoftwareTitleAutoUpdateConfig(ctx context.Context, titleID uint, teamID *uint, config fleet.SoftwareAutoUpdateConfig) error {
	if err := svc.authz.Authorize(ctx, &fleet.VPPApp{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// Coerce nil teamID to 0.
	var tID uint
	if teamID != nil {
		tID = *teamID
	}
	if err := svc.ds.UpdateSoftwareTitleAutoUpdateConfig(ctx, titleID, tID, config); err != nil {
		return ctxerr.Wrap(ctx, err, "updating software title auto update config")
	}

	return nil
}
