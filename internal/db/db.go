package db

import (
	"context"
	"fmt"
	"time"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ESIndex struct {
	gorm.Model
	Name          string `gorm:"type:varchar(255);not null;uniqueIndex:uk_es_index_name"`
	IndexCreateAt TimeString
	StoreSize     string
}

type ESSnapshot struct {
	gorm.Model
	Snapshot   string `gorm:"type:varchar(255);not null;uniqueIndex:uk_es_snapshot_name"`
	Repository string
	State      string
	StartTime  TimeString
	Indices    datatypes.JSON
}

type Task struct {
	gorm.Model
	TaskID       string `gorm:"size:64;index;not null"`
	Index        string `gorm:"index;not null"`
	Repository   string
	Snapshot     string
	Status       string  `gorm:"size:20;index;not null"` // PENDING, RUNNING, SUCCESS, FAILED, TIMEOUT, CANCELED
	CurrentStage *string `gorm:"size:32"`
	Payload      *string `gorm:"type:json"`
	ErrorMessage *string `gorm:"type:text"`

	StartedAt  *time.Time
	FinishedAt *time.Time
}

func NewDB(lc fx.Lifecycle) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if config.GlobalConfig.DB.Host != "" {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.GlobalConfig.DB.Username,
			config.GlobalConfig.DB.Password,
			config.GlobalConfig.DB.Host,
			config.GlobalConfig.DB.Port,
			config.GlobalConfig.DB.Name,
		)

		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Error().Err(err).Msgf("failed to connect to mysql %s:%d db %s with user %s", config.GlobalConfig.DB.Host, config.GlobalConfig.DB.Port, config.GlobalConfig.DB.Name, config.GlobalConfig.DB.Username)
			return nil, err
		}
	} else {
		db, err = gorm.Open(sqlite.Open(fmt.Sprintf("%s.db", config.GlobalConfig.DB.Name)), &gorm.Config{})
		if err != nil {
			log.Error().Err(err).Msgf("failed to connect to sqlite %s.db", config.GlobalConfig.DB.Name)
			return nil, err
		}
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info().Msg("db start")
			if err := db.AutoMigrate(&ESIndex{}, &ESSnapshot{}, &Task{}); err != nil {
				log.Error().Err(err).Msg("failed to migrate db")
				return err
			}
			return nil
		},
		OnStop: func(context.Context) error {
			sqldb, err := db.DB()
			if err != nil {
				return err
			}

			if err := sqldb.Close(); err != nil {
				log.Error().Err(err).Msg("failed to close db connection")
				return err
			}

			return nil
		},
	})

	return db, nil
}

// Create records in batch but with no conflict
func CreateRecords[T any](db *gorm.DB, records *[]T) error {
	return db.Create(records).Error
}

// Create records in batch, if onconflict on name(uniq index) column, then update the store_size and updated_at column
func CreateIndexRecords[T any](db *gorm.DB, records *[]T) error {
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"store_size": gorm.Expr("VALUES(store_size)"),
			"updated_at": gorm.Expr("VALUES(updated_at)"),
		}),
	}).Create(records).Error

	return err
}

// Create records in batch, if onconflict on name(uniq index) column, then update the store_size and updated_at column
func CreateSnapshotRecords[T any](db *gorm.DB, records *[]T) error {
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"Snapshot":   gorm.Expr("VALUES(snapshot)"),
			"updated_at": gorm.Expr("VALUES(updated_at)"),
		}),
	}).Create(records).Error

	return err
}

// Query all records from db to meet conds and order
func QueryAll[T any](db *gorm.DB, order string, limit int, conds ...any) ([]T, error) {
	var records []T
	tx := db

	if len(conds) > 0 {
		tx = tx.Where(conds[0], conds[1:]...)
	}

	if order != "" {
		tx = tx.Order(order)
	}

	if limit != 0 {
		tx = tx.Limit(limit)
	}

	result := tx.Debug().Find(&records)

	return records, result.Error
}

// Delete record from db to meet conds
func DeleteRecord[T any](db *gorm.DB, record *T, conds ...any) error {
	tx := db

	if len(conds) > 0 {
		tx = tx.Where(conds[0], conds[1:]...)
	}

	result := tx.Delete(record)

	return result.Error
}
