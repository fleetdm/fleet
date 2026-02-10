package activity

// DataProviders combines providers for ACL layer.
type DataProviders interface {
	UserProvider
	HostProvider
}
