package mock

//go:generate go run ../../mock/mockimpl/impl.go -o service_osquery.go "s *TLSService" "fleet.OsqueryService"
//go:generate go run ../../mock/mockimpl/impl.go -o service_pusher_factory.go "s *APNSPushProviderFactory" "github.com/micromdm/nanomdm/push.PushProviderFactory"
//go:generate go run ../../mock/mockimpl/impl.go -o service_push_provider.go "s *APNSPushProvider" "github.com/micromdm/nanomdm/push.PushProvider"
