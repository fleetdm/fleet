package fleet

type Transactions interface {
	Begin() (Transaction, error)
}

type Transaction interface {
	Commit() error
	Rollback() error
}

func HasTransaction(tx Transaction) OptionalArg {
	return func() interface{} {
		return tx
	}
}
