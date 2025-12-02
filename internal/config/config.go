package config

import (
	"log"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr    string
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
	JWTLeeway   time.Duration

	Kafka struct {
		Brokers []string
		Topic   string
		Group   string
	}
}

func Load() Config {
	cfg := Config{}

	cfg.HTTPAddr = getEnv("HTTP_ADDR", ":8080")
	cfg.JWTSecret = getEnv("JWT_SECRET", "devsecret")
	cfg.JWTIssuer = getEnv("JWT_ISSUER", "")
	cfg.JWTAudience = getEnv("JWT_AUDIENCE", "")
	cfg.JWTLeeway = durationPars("JWT_LEEWAY", 5*time.Second)

	brokersRaw := getEnv("KAFKA_BROKERS", "localhost:9092")
	cfg.Kafka.Brokers = splitAndTrim(brokersRaw)
	cfg.Kafka.Topic = getEnv("KAFKA_TOPIC", "ws-messages")
	cfg.Kafka.Group = getEnv("KAFKA_GROUP", "ws-dispatcher")

	return cfg
}

func getEnv(key, def string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}
	return value
}

func durationPars(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid duration for %s: %v, using default %s", key, err, def)
		return def
	}
	return d
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
