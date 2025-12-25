package cmd

import (
	"github.com/404LifeFound/es-snapshot-restore/internal/cache"
	"github.com/404LifeFound/es-snapshot-restore/internal/cron"
	"github.com/404LifeFound/es-snapshot-restore/internal/db"
	"github.com/404LifeFound/es-snapshot-restore/internal/elastic"
	"github.com/404LifeFound/es-snapshot-restore/internal/http"
	"github.com/404LifeFound/es-snapshot-restore/internal/k8s"
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
					k8s.NewClient,
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

	// flags for cronjob
	flags.String("cron-schedule", "0 */10 * * * *", "cron job schedule")

	//flags for es
	flags.String("es-restorekey", "restore", "restore attr key")
	flags.Int32("es-restorecount", 2, "restore nodeset count")
	flags.String("es-serviceaccount", "elastic-search", "Elasticsearch serviceaccount name")
	flags.StringSlice("es-plugins", []string{"mapper-size", "repository-gcs"}, "Elasticsearch plugins")
	flags.String("es-requestcpu", "4", "request cpu resource")
	flags.String("es-requestmem", "8", "request mem resource")
	flags.String("es-limitcpu", "4", "limit mem resource")
	flags.String("es-limitmem", "8", "limit mem resource")
	flags.String("es-storageclass", "standard-rwo", "storage class name")
	flags.String("es-containername", "elasticsearch", "elasticsearch container name")
	flags.String("es-topologykey", "kubernetes.io/hostname", "elasticsearch topology key")
	flags.StringToString("es-labels", map[string]string{}, "es labels")
	flags.StringToString("es-annotations", map[string]string{}, "es annotations")
	flags.StringToString("es-tolerations", map[string]string{}, "es tolerations")

	//flags for kubernetes
	flags.String("kube-config", "~/.kube/config", "kubeconfig file path")

	return serverCmd
}
