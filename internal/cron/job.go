package cron

import (
	"context"
	"encoding/json"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/404LifeFound/es-snapshot-restore/internal/db"
	"github.com/404LifeFound/es-snapshot-restore/internal/elastic"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AllIndex struct {
	ES       *elastic.ES
	DBClient *gorm.DB
}

func (a *AllIndex) Run() {
	var all_index []db.ESIndex
	ctx := context.Background()
	indexs, err := a.ES.GetAllIndex(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all index from elasticsearch")
	}

	for _, i := range indexs {
		index_create_time, err := db.NewTimeString(i.CreateAt)
		if err != nil {
			log.Error().Err(err).Msg("faild to parse time string to TimeString")
		}
		all_index = append(all_index, db.ESIndex{
			Name:          i.Name,
			IndexCreateAt: index_create_time,
			StoreSize:     i.StoreSize,
		})
	}

	if err := db.CreateIndexRecords[db.ESIndex](a.DBClient, &all_index); err != nil {
		log.Error().Err(err).Msg("failed to create all es index records")
	} else {
		log.Info().Msg("create all index records success")
	}
}

type AllSnapshot struct {
	ES       *elastic.ES
	DBClient *gorm.DB
}

func (a *AllSnapshot) Run() {
	var all_snapshots []db.ESSnapshot
	ctx := context.Background()
	snapshots, err := a.ES.GetAllSnapshotDetails(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all snapshots from elasticsearch")
	}

	for _, s := range snapshots {
		snapshot_create_time, err := db.NewTimeString(s.StartTime)
		if err != nil {
			log.Error().Err(err).Msg("faild to parse time string to TimeString")
		}

		indices, err := json.Marshal(s.Indices)
		if err != nil {
			log.Error().Err(err).Msg("faild to parse marshal snapshot indices")
		}
		all_snapshots = append(all_snapshots, db.ESSnapshot{
			Snapshot:   s.Snapshot,
			Repository: s.Repository,
			State:      s.State,
			StartTime:  snapshot_create_time,
			Indices:    datatypes.JSON(indices),
		})
	}

	if err := db.CreateSnapshotRecords[db.ESSnapshot](a.DBClient, &all_snapshots); err != nil {
		log.Error().Err(err).Msg("failed to create all es snapshots records")
	} else {
		log.Info().Msg("create all snaphosts records success")
	}
}

func RegisterJobs(lc fx.Lifecycle, c *cron.Cron, es *elastic.ES, db *gorm.DB) {
	all_index_job := &AllIndex{
		ES:       es,
		DBClient: db,
	}

	all_snapshot_job := &AllSnapshot{
		ES:       es,
		DBClient: db,
	}
	c.AddJob(config.GlobalConfig.Cron.Schedule, all_index_job)
	c.AddJob(config.GlobalConfig.Cron.Schedule, all_snapshot_job)
}
