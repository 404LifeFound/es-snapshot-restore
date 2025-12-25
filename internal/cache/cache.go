package cache

import (
	"fmt"
	"time"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/metrics"
	"github.com/eko/gocache/lib/v4/store"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"
	rueidis_store "github.com/eko/gocache/store/rueidis/v4"
	"github.com/redis/rueidis"
)

func NewCache() (cache.CacheInterface[string], error) {
	var s store.StoreInterface

	if config.GlobalConfig.Redis.Host != "" {
		rueidisClient, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress: []string{fmt.Sprintf("%s:%d", config.GlobalConfig.Redis.Host, config.GlobalConfig.Redis.Port)},
			Password:    config.GlobalConfig.Redis.Password,
			SelectDB:    config.GlobalConfig.Redis.DB,
		})
		if err != nil {
			return nil, err
		}

		s = rueidis_store.NewRueidis(rueidisClient, store.WithClientSideCaching(15*time.Second))
	} else {
		ristrettoCache, err := ristretto.NewCache(&ristretto.Config[string, any]{
			NumCounters: 1000,
			MaxCost:     100,
			BufferItems: 64,
		})
		if err != nil {
			return nil, err
		}

		s = ristretto_store.NewRistretto(ristrettoCache)
	}

	p := metrics.NewPrometheus("es-snapshot-restore")
	c := cache.New[string](s)

	return cache.NewMetric(p, c), nil
}
