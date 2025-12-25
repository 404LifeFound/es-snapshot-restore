package cmd

import (
	"github.com/404LifeFound/es-snapshot-restore/internal/cache"
	"github.com/404LifeFound/es-snapshot-restore/internal/cron"
	"github.com/404LifeFound/es-snapshot-restore/internal/db"
	"github.com/404LifeFound/es-snapshot-restore/internal/elastic"
	"github.com/404LifeFound/es-snapshot-restore/internal/http"
	"github.com/ipfans/fxlogger"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func NewServerCmd() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "run server mode",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msgf("run: %s", cmd.Name())
			app := fx.New(
				fx.Provide(
					cache.NewCache,
					http.NewGinEngine,
					db.NewDB,
					elastic.NewDefaultESConfig,
					elastic.NewES,
					cron.NewCron,
				),
				fx.Invoke(
					http.RegisterHandler,
					cron.RegisterJobs,
				),
				fx.WithLogger(fxlogger.WithZerolog(log.Logger)),
			)
			app.Run()
		},
	}

	flags := serverCmd.Flags()

	//flag for http server
	flags.String("http-host", "127.0.0.1", "http host")
	flags.Int("http-port", 8080, "http port")
	flags.Bool("http-releasemode", false, "run http server on release mode")

	// flags for kibana
	flags.String("kibana-host", "127.0.0.1", "kibana host")
	flags.Int("kibana-port", 5601, "kibana port")

	// flags for db(mysql)
	flags.String("db-host", "", "db host")
	flags.Int("db-port", 3306, "db port")
	flags.String("db-username", "", "db username")
	flags.String("db-password", "", "db password")
	flags.String("db-name", "es_snapshot_restore", "db name")

	// flags for cache(redis)
	flags.String("redis-host", "127.0.0.1", "redis host")
	flags.Int("redis-port", 6379, "redis port")
	flags.String("redis-password", "", "redis password")
	flags.Int("redis-db", 0, "redis db")

	// flags from cronjob
	flags.String("cron-schedule", "0 */10 * * * *", "cron job schedule")

	return serverCmd
}
