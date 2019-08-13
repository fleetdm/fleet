package kolide

// service a interface stub
type Service interface {
	UserService
	SessionService
	PackService
	LabelService
	QueryService
	CampaignService
	OsqueryService
	OsqueryOptionsService
	HostService
	AppConfigService
	InviteService
	TargetService
	ScheduledQueryService
	OptionService
	FileIntegrityMonitoringService
	StatusService
}
