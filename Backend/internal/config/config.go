package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func Load(envFile string) error {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return fmt.Errorf("gagal membaca file environment: %w", err)
		}
	}

	required := []string{
		"PGHOST",
		"PGPORT",
		"PGUSER",
		"PGPASSWORD",
		"PGDATABASE",
		"PGSSLMODE",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return fmt.Errorf("konfigurasi %s belum diisi", key)
		}
	}

	return nil
}
