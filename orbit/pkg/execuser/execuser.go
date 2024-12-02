// Package execuser is used to run applications from a high privilege user (root on Unix,
// SYSTEM service on Windows) as the current login user.
package execuser

import "io"

type eopts struct {
	env        [][2]string
	args       [][2]string
	stderrPath string //nolint:structcheck,unused
}

// Option allows configuring the application.
type Option func(*eopts)

// WithEnv sets environment variables for the application.
func WithEnv(name, value string) Option {
	return func(a *eopts) {
		a.env = append(a.env, [2]string{name, value})
	}
}

// WithArg sets command line arguments for the application.
func WithArg(name, value string) Option {
	return func(a *eopts) {
		a.args = append(a.args, [2]string{name, value})
	}
}

// Run runs an application as the current login user.
// It assumes the caller is running with high privileges (root on Unix, SYSTEM on Windows).
//
// It returns after starting the child process.
func Run(path string, opts ...Option) (lastLogs string, err error) {
	var o eopts
	for _, fn := range opts {
		fn(&o)
	}
	return run(path, o)
}

// RunWithOutput runs an application as the current login user and returns its output.
// It assumes the caller is running with high privileges (root on UNIX).
//
// It blocks until the child process exits.
// Non ExitError errors return with a -1 exitCode.
func RunWithOutput(path string, opts ...Option) (output []byte, exitCode int, err error) {
	var o eopts
	for _, fn := range opts {
		fn(&o)
	}
	return runWithOutput(path, o)
}

func RunWithStdin(path string, opts ...Option) (io.WriteCloser, error) {
	var o eopts
	for _, fn := range opts {
		fn(&o)
	}
	return runWithStdin(path, o)
}
