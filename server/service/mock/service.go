package mock

//go:generate mockimpl -o service_osquery.go "s *TLSService" "fleet.OsqueryService"
//go:generate mockimpl -o service_pusher_factory.go "s *APNSPushProviderFactory" "github.com/micromdm/nanomdm/push.PushProviderFactory"
//go:generate mockimpl -o service_push_provider.go "s *APNSPushProvider" "github.com/micromdm/nanomdm/push.PushProvider"
