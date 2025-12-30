package utils

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func RandomString(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}

func PtrToAny[T any](s T) *T {
	return &s
}

func RandomName() string {
	name := fmt.Sprintf("%s-%s", config.GlobalConfig.ES.RestoreKey, RandomString(config.GlobalConfig.ES.RandomLen))
	log.Info().Msgf("restore node name is: %s", name)
	return name
}

func TaskID() string {
	return uuid.New().String()
}

func ToGB(sizeStr string) (float64, error) {
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
	case "", "B":
		return value / (1024 * 1024 * 1024), nil
	case "K", "KB", "KI":
		return value / (1024 * 1024), nil
	case "M", "MB", "MI":
		return value / 1024, nil
	case "G", "GB", "GI":
		return value, nil
	case "T", "TB", "TI":
		return value * 1024, nil
	case "P", "PB", "PI":
		return value * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknow unit: %s", unit)
	}
}
