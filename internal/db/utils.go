package db

import (
	"github.com/404LifeFound/es-snapshot-restore/internal/utils"
	"github.com/rs/zerolog/log"
)

type ESIndexs []ESIndex

func (e *ESIndexs) StoreSize() float64 {
	var totalGB float64
	for _, i := range *e {
		gb, err := utils.ToGB(i.StoreSize)
		log.Debug().Msgf("index %s store size is: %.6f", i.Name, gb)
		if err != nil {
			log.Error().Err(err).Msgf("faild to parse index %s store size: %s to float GB", i.Name, i.StoreSize)
			continue
		}

		totalGB += gb
	}
	log.Info().Msgf("total store size is: %.6f", totalGB)

	return totalGB
}

func (e *ESIndexs) IndexNames() []string {
	var n []string
	for _, i := range *e {
		n = append(n, i.Name)
	}

	return n
}
