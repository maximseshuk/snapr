package notifier

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/logger"
)

type Event struct {
	JobName  string
	Success  bool
	Duration string
	Error    string
}

type Notifier interface {
	GetType() string
	Notify(ctx context.Context, ev Event) error
}

type Dispatcher struct {
	notifiers []boundNotifier
	systemLog zerolog.Logger
}

type boundNotifier struct {
	cfg config.NotifierConfig
	n   Notifier
}

func NewDispatcher(configs []config.NotifierConfig) *Dispatcher {
	d := &Dispatcher{systemLog: logger.NewSystemLogger("notifier")}
	for _, c := range configs {
		n, err := build(c)
		if err != nil {
			d.systemLog.Error().Err(err).Str("type", c.Type).Str("name", c.Name).Msg("Skipping notifier")
			continue
		}
		d.notifiers = append(d.notifiers, boundNotifier{cfg: c, n: n})
	}
	return d
}

func (d *Dispatcher) Dispatch(ctx context.Context, ev Event) {
	for _, b := range d.notifiers {
		if ev.Success && !b.cfg.OnSuccess {
			continue
		}
		if !ev.Success && !b.cfg.OnFailure {
			continue
		}
		if err := b.n.Notify(ctx, ev); err != nil {
			d.systemLog.Error().Err(err).
				Str("type", b.cfg.Type).
				Str("name", b.cfg.Name).
				Str("job", ev.JobName).
				Msg("Notifier delivery failed")
		}
	}
}

func build(c config.NotifierConfig) (Notifier, error) {
	switch c.Type {
	case "webhook":
		return NewWebhook(c)
	case "telegram":
		return NewTelegram(c)
	case "email":
		return NewEmail(c)
	}
	return nil, fmt.Errorf("unknown notifier type: %s", c.Type)
}
