package util

import (
	"fmt"
	"time"
)

// StartSpinner prints an inline spinner until the returned stop function is called.
func StartSpinner(msg string) func() {
	done := make(chan struct{})
	go func() {
		chars := []rune{'|', '/', '-', '\\'}
		i := 0
		for {
			select {
			case <-done:
				fmt.Printf("\r%s... done.\n", msg)
				return
			default:
				fmt.Printf("\r%s %c", msg, chars[i%len(chars)])
				i++
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	return func() { close(done) }
}
