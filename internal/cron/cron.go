package cron

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

func NewCron(lc fx.Lifecycle) *cron.Cron {
	logger := &logger{}
	c := cron.New(
		cron.WithParser(cron.NewParser(
			cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor,
		)),
		cron.WithLogger(logger),
		cron.WithChain(cron.SkipIfStillRunning(logger), cron.Recover(logger)),
		cron.WithLocation(time.Local),
	)
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			c.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			select {
			case <-c.Stop().Done():
				log.Debug().Msg("Cron stopped")
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})

	return c
}
