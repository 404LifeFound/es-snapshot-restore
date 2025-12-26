package utils

import (
	"fmt"
	"math/rand"
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
