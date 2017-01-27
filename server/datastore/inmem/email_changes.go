package inmem

func (ds *Datastore) PendingEmailChange(uid uint, newEmail, token string) error {
	panic("deprecated")
}

func (ds *Datastore) ConfirmPendingEmailChange(token string) (string, error) {
	panic("deprecated")
}
