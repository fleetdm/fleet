package kolide

// service a interface stub
type Service interface {
	UserService
	SessionService
	PackService
	QueryService
	OsqueryService
	HostService
	AppConfigService
	InviteService
}
