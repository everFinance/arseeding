package arseeding

import (
	"github.com/getsentry/sentry-go"
	"github.com/inconshreveable/log15"
)

func NewLog(serverName string) log15.Logger {
	lg := log15.New("module", serverName)

	// 默认的 logger handle
	h := lg.GetHandler()
	// 集成 sentry 的 logger handle
	sentryHandle := log15.FuncHandler(func(r *log15.Record) error {
		if r.Lvl == log15.LvlError {
			msg := string(log15.JsonFormat().Format(r))
			go func(m string) {
				sentry.CaptureMessage(m)
			}(msg)
		}
		return nil
	})

	lg.SetHandler(log15.MultiHandler(h, sentryHandle))

	return lg
}
