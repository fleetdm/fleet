package fleet

type AddFleetMaintainedAppRequest struct {
	TeamID            *uint    `json:"team_id" renameto:"fleet_id"`
	AppID             uint     `json:"fleet_maintained_app_id"`
	InstallScript     string   `json:"install_script"`
	PreInstallQuery   string   `json:"pre_install_query"`
	PostInstallScript string   `json:"post_install_script"`
	SelfService       bool     `json:"self_service"`
	UninstallScript   string   `json:"uninstall_script"`
	LabelsIncludeAny  []string `json:"labels_include_any"`
	LabelsExcludeAny  []string `json:"labels_exclude_any"`
	AutomaticInstall  bool     `json:"automatic_install"`
	Categories        []string `json:"categories"`
}

type AddFleetMaintainedAppResponse struct {
	SoftwareTitleID uint  `json:"software_title_id,omitempty"`
	Err             error `json:"error,omitempty"`
}

func (r AddFleetMaintainedAppResponse) Error() error { return r.Err }

type ListFleetMaintainedAppsRequest struct {
	ListOptions
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type ListFleetMaintainedAppsResponse struct {
	FleetMaintainedApps []MaintainedApp     `json:"fleet_maintained_apps"`
	Meta                *PaginationMetadata `json:"meta"`
	Err                 error               `json:"error,omitempty"`
}

func (r ListFleetMaintainedAppsResponse) Error() error { return r.Err }

type GetFleetMaintainedAppRequest struct {
	AppID  uint  `url:"app_id"`
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetFleetMaintainedAppResponse struct {
	FleetMaintainedApp *MaintainedApp `json:"fleet_maintained_app"`
	Err                error          `json:"error,omitempty"`
}

func (r GetFleetMaintainedAppResponse) Error() error { return r.Err }
