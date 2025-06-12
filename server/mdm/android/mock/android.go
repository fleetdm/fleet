package mock

//go:generate go run ../../../mock/mockimpl/impl.go -o proxy.go "p *Proxy" "android.Proxy"
//go:generate go run ../../../mock/mockimpl/impl.go -o datastore.go "ds *Datastore" "android.Datastore"
