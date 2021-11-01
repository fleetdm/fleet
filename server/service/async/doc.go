// Package async implements service tasks that are to be processed
// asynchronously by storing the data temporarily in Redis on the hot path and
// at regular intervals, efficiently moving that data to the persistent mysql
// database.  This pattern allows to avoid the thundering herd problem in
// setups with lots of hosts, by collecting the data in fast storage (Redis)
// and then using a background task to store it down to persistent storage
// (mysql) in a controlled manner.
//
// TODO: for now, only label queries are done like this - if it proves to be
// efficient and manageable, policy queries and saving of hosts could use
// this approach too.
package async
