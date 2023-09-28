//go:build unix

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/oklog/run"
	"github.com/rs/zerolog/log"
)

func signalHandler(ctx context.Context) (execute func() error, interrupt func(error)) {
	return run.SignalHandler(ctx, os.Interrupt, syscall.SIGTERM)
}

func sigusrListener(rootDir string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	for {
		<-c

		if err := dumpProf(rootDir); err != nil {
			log.Warn().Err(err).Msg("unable to create pprof")
		}
	}
}

func dumpProf(rootDir string) error {
	if err := secure.MkdirAll(path.Join(rootDir, "profiles"), constant.DefaultDirMode); err != nil {
		return err
	}
	now := time.Now()

	// We can't use ISO 8601/RFC 3339 because NTFS and FAT do not allow colons in filenames
	timestamp := now.UTC().Format("2006-01-02T15-04-05")

	out, err := os.Create(path.Join(rootDir, "profiles", fmt.Sprintf("profiles-%s.tar.gz", timestamp)))
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	buf := new(bytes.Buffer)

	for _, profile := range pprof.Profiles() {
		err = profile.WriteTo(buf, 0)
		if err != nil {
			return err
		}

		header := tar.Header{
			Typeflag:   tar.TypeReg,
			Name:       fmt.Sprintf("%s.pprof", profile.Name()),
			Size:       int64(buf.Len()),
			Mode:       0o664,
			ModTime:    now,
			AccessTime: now,
			ChangeTime: now,
		}
		err = tw.WriteHeader(&header)
		if err != nil {
			return err
		}
		_, err = buf.WriteTo(tw)
		if err != nil {
			return err
		}
		buf.Reset()
	}
	return nil
}
