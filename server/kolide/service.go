package kolide

// service a interface stub
type Service interface {
	UserService
	SessionService
	PackService
	LabelService
	QueryService
	OsqueryService
	HostService
	AppConfigService
	InviteService
	TargetService
}
