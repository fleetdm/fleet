package mock

//go:generate go run ../../../mock/mockimpl/impl.go -o client.go "p *Client" "androidmgmt.Client"
//go:generate go run ../../../mock/mockimpl/impl.go -o datastore.go "ds *Datastore" "android.Datastore"
