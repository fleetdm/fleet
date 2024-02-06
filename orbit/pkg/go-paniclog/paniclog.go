package paniclog

import "os"

// UndoFunction will reverse the redirection
type UndoFunction func() error

// RedirectStderr to the file passed in, so that the output of any panics that
// occur will be sent to that file. The caller may close the file after
// this function returns.
//
// Of course, anything else written to stderr will also be sent to that file,
// so don't do that unless that's your intent.
//
// Returns a function that can be used to revert stderr back to the console,
// or an error if stderr could not be redirected
func RedirectStderr(f *os.File) (UndoFunction, error) {
	return redirectStderr(f)
}
