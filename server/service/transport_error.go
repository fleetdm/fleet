package service

// errorer interface is implemented by response structs to encode business logic errors
type errorer interface {
	Error() error
}
