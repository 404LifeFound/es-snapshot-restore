package db

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type ESIndexs []ESIndex

func toGB(sizeStr string) (float64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	re := regexp.MustCompile(`(?i)^([\d.]+)\s*([a-zA-Z]+)$`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		log.Error().Err(fmt.Errorf("can't parse: %s", sizeStr))
		return 0, fmt.Errorf("can't parse: %s", sizeStr)
	}

	log.Debug().Msgf("match[0] is %s,match[1] is %s,match[2] is %s", matches[0], matches[1], matches[2])

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		log.Error().Err(err).Msgf("can't parse string to float 64: %s", matches[1])
		return 0, err
	}

	unit := strings.ToUpper(matches[2])

	switch unit {
	case "B":
		return value / (1024 * 1024 * 1024), nil
	case "KB":
		return value / (1024 * 1024), nil
	case "MB":
		return value / 1024, nil
	case "GB":
		return value, nil
	case "TB":
		return value * 1024, nil
	case "PB":
		return value * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknow unit: %s", unit)
	}
}

func (e *ESIndexs) StoreSize() float64 {
	var totalGB float64
	for _, i := range *e {
		gb, err := toGB(i.StoreSize)
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
