package condaccess

import (
	"fmt"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// kitlogAdapter adapts kitlog.Logger to saml logger.Interface
type kitlogAdapter struct {
	logger kitlog.Logger
}

func (k *kitlogAdapter) Printf(format string, v ...interface{}) {
	level.Info(k.logger).Log("msg", fmt.Sprintf(format, v...))
}

func (k *kitlogAdapter) Print(v ...interface{}) {
	level.Info(k.logger).Log("msg", fmt.Sprint(v...))
}

func (k *kitlogAdapter) Println(v ...interface{}) {
	level.Info(k.logger).Log("msg", fmt.Sprint(v...))
}

func (k *kitlogAdapter) Fatal(v ...interface{}) {
	level.Error(k.logger).Log("msg", fmt.Sprint(v...))
}

func (k *kitlogAdapter) Fatalf(format string, v ...interface{}) {
	level.Error(k.logger).Log("msg", fmt.Sprintf(format, v...))
}

func (k *kitlogAdapter) Fatalln(v ...interface{}) {
	level.Error(k.logger).Log("msg", fmt.Sprint(v...))
}

func (k *kitlogAdapter) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	level.Error(k.logger).Log("msg", msg)
	panic(msg)
}

func (k *kitlogAdapter) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	level.Error(k.logger).Log("msg", msg)
	panic(msg)
}

func (k *kitlogAdapter) Panicln(v ...interface{}) {
	msg := fmt.Sprint(v...)
	level.Error(k.logger).Log("msg", msg)
	panic(msg)
}
