package command

// seederLogger adapts the package-level printf helper to the seed.Logger
// interface, so seed/ doesn't need to import this package.
type seederLogger struct{}

func (seederLogger) Printf(format string, a ...any) {
	printf(format, a...)
}
