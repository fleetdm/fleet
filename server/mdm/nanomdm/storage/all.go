package storage

// AllStorage represents all required storage by NanoMDM
type AllStorage interface {
	ServiceStore
	PushStore
	PushCertStore
	CommandEnqueuer
	CertAuthStore
	CertAuthRetriever
	StoreMigrator
	TokenUpdateTallyStore
}
