package condaccess

import (
	"context"
	"fmt"
	"log/slog"
)

// slogAdapter adapts *slog.Logger to saml logger.Interface
type slogAdapter struct {
	ctx    context.Context
	logger *slog.Logger
}

func (k *slogAdapter) Printf(format string, v ...any) {
	k.logger.InfoContext(k.ctx, fmt.Sprintf(format, v...))
}

func (k *slogAdapter) Print(v ...any) {
	k.logger.InfoContext(k.ctx, fmt.Sprint(v...))
}

func (k *slogAdapter) Println(v ...any) {
	k.logger.InfoContext(k.ctx, fmt.Sprint(v...))
}

func (k *slogAdapter) Fatal(v ...any) {
	k.logger.ErrorContext(k.ctx, fmt.Sprint(v...))
}

func (k *slogAdapter) Fatalf(format string, v ...any) {
	k.logger.ErrorContext(k.ctx, fmt.Sprintf(format, v...))
}

func (k *slogAdapter) Fatalln(v ...any) {
	k.logger.ErrorContext(k.ctx, fmt.Sprint(v...))
}

func (k *slogAdapter) Panic(v ...any) {
	msg := fmt.Sprint(v...)
	k.logger.ErrorContext(k.ctx, msg)
	panic(msg)
}

func (k *slogAdapter) Panicf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	k.logger.ErrorContext(k.ctx, msg)
	panic(msg)
}

func (k *slogAdapter) Panicln(v ...any) {
	msg := fmt.Sprint(v...)
	k.logger.ErrorContext(k.ctx, msg)
	panic(msg)
}
