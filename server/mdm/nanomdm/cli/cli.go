// Package cli contains shared command-line helpers and utilities.
package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/allmulti"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/file"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/mysql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/micromdm/nanolib/log"
)

type StringAccumulator []string

func (s *StringAccumulator) String() string {
	return strings.Join(*s, ",")
}

func (s *StringAccumulator) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type Storage struct {
	Storage StringAccumulator
	DSN     StringAccumulator
	Options StringAccumulator
}

func NewStorage() *Storage {
	return &Storage{}
}

func (s *Storage) Parse(logger log.Logger) (storage.AllStorage, error) {
	if len(s.Storage) != len(s.DSN) {
		return nil, errors.New("must have same number of storage and DSN flags")
	}
	if len(s.Options) > 0 && len(s.Storage) != len(s.Options) {
		return nil, errors.New("must have same number of storage and storage options flags")
	}
	// default storage and DSN pair
	if len(s.Storage) < 1 {
		s.Storage = append(s.Storage, "file")
		s.DSN = append(s.DSN, "db")
	}
	var mdmStorage []storage.AllStorage
	for idx, storage := range s.Storage {
		dsn := s.DSN[idx]
		options := ""
		if len(s.Options) > 0 {
			options = s.Options[idx]
		}
		logger.Info(
			"msg", "storage setup",
			"storage", storage,
		)
		switch storage {
		case "file":
			fileStorage, err := fileStorageConfig(dsn, options)
			if err != nil {
				return nil, err
			}
			mdmStorage = append(mdmStorage, fileStorage)
		case "mysql":
			mysqlStorage, err := mysqlStorageConfig(dsn, options, logger)
			if err != nil {
				return nil, err
			}
			mdmStorage = append(mdmStorage, mysqlStorage)
		default:
			return nil, fmt.Errorf("unknown storage: %s", storage)
		}
	}
	if len(mdmStorage) < 1 {
		return nil, errors.New("no storage setup")
	}
	if len(mdmStorage) == 1 {
		return mdmStorage[0], nil
	}
	logger.Info("msg", "storage setup", "storage", "multi-storage", "count", len(mdmStorage))
	return allmulti.New(
		logger.With("component", "multi-storage"),
		mdmStorage...,
	), nil
}

var NoStorageOptions = errors.New("storage backend does not support options, please specify no (or empty) options")

func fileStorageConfig(dsn, options string) (*file.FileStorage, error) {
	if options != "" {
		return nil, NoStorageOptions
	}
	return file.New(dsn)
}

func mysqlStorageConfig(dsn, options string, logger log.Logger) (*mysql.MySQLStorage, error) {
	logger = logger.With("storage", "mysql")
	// mysql.WithLogger requires *slog.Logger; bridge the nanolib logger
	slogLogger := slog.New(&nanoLibSlogHandler{logger: logger})
	opts := []mysql.Option{
		mysql.WithDSN(dsn),
		mysql.WithLogger(slogLogger),
	}
	if options != "" {
		for k, v := range splitOptions(options) {
			switch k {
			case "delete":
				if v == "1" {
					opts = append(opts, mysql.WithDeleteCommands())
					logger.Debug("msg", "deleting commands")
				} else if v != "0" {
					return nil, fmt.Errorf("invalid value for delete option: %q", v)
				}
			default:
				return nil, fmt.Errorf("invalid option: %q", k)
			}
		}
	}
	return mysql.New(opts...)
}

func splitOptions(s string) map[string]string {
	out := make(map[string]string)
	opts := strings.Split(s, ",")
	for _, opt := range opts {
		optKAndV := strings.SplitN(opt, "=", 2)
		if len(optKAndV) < 2 {
			optKAndV = append(optKAndV, "")
		}
		out[optKAndV[0]] = optKAndV[1]
	}
	return out
}

// nanoLibSlogHandler adapts a nanolib/log.Logger to slog.Handler.
// This bridge exists because the standalone nanomdm CLI tools still use
// nanolib loggers, while the mysql storage backend now uses *slog.Logger.
type nanoLibSlogHandler struct {
	logger log.Logger
	attrs  []slog.Attr
}

func (h *nanoLibSlogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *nanoLibSlogHandler) Handle(_ context.Context, r slog.Record) error {
	kvs := make([]any, 0, 2+2*len(h.attrs)+2*r.NumAttrs())
	kvs = append(kvs, "msg", r.Message)
	for _, a := range h.attrs {
		kvs = append(kvs, a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		kvs = append(kvs, a.Key, a.Value.Any())
		return true
	})
	if r.Level >= slog.LevelInfo {
		h.logger.Info(kvs...)
	} else {
		h.logger.Debug(kvs...)
	}
	return nil
}

func (h *nanoLibSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &nanoLibSlogHandler{logger: h.logger, attrs: newAttrs}
}

func (h *nanoLibSlogHandler) WithGroup(_ string) slog.Handler {
	return h
}
