//go:build darwin || !cgo
// +build darwin !cgo

package table

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"
)

func mdfind(args ...string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := "/usr/bin/mdfind"

	out, err := exec.CommandContext(ctx, path, args...).Output()
	if err != nil {
		return nil, err
	}
	var lines []string
	lr := bufio.NewReader(bytes.NewReader(out))
	for {
		line, _, err := lr.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}
