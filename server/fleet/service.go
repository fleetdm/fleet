package fleet

// service a interface stub
type Service interface {
	UserService
	SessionService
	PackService
	LabelService
	QueryService
	CampaignService
	OsqueryService
	AgentOptionsService
	HostService
	AppConfigService
	InviteService
	TargetService
	ScheduledQueryService
	StatusService
	CarveService
	TeamService
	ActivitiesService
	UserRolesService
	GlobalScheduleService
	TranslatorService
	TeamScheduleService
}
