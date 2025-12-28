package config

import (
	"os"
)

// Config groups runtime configuration for the backend service.
type Config struct {
	GRPCPort                 string
	DbUrl                    string
	JwtSecretKey             []byte
	BotUsername              string
	BotToken                 string
	ChatID                   string
	SkipChatMembershipCheck  bool
}

// Load reads configuration from environment with sane defaults.
func Load() Config {
	port := getenv("GRPC_PORT", "6969")
	url := getenv("POSTGRES_URL", "postgres://user:password@localhost:5432/musicclubbot")
	jwtSecret := []byte(getenv("JWT_SECRET", "change-this-in-prod"))
	botUsername := getenv("BOT_USERNAME", "YourBotUsername")
	botToken := getenv("BOT_TOKEN", "")
	chatID := getenv("CHAT_ID", "")
	skipCheck := getenv("SKIP_CHAT_MEMBERSHIP_CHECK", "false") == "true"
	
	return Config{
		GRPCPort:                port,
		DbUrl:                   url,
		JwtSecretKey:            jwtSecret,
		BotUsername:             botUsername,
		BotToken:                botToken,
		ChatID:                  chatID,
		SkipChatMembershipCheck: skipCheck,
	}
}

func (c Config) GRPCAddr() string {
	return ":" + c.GRPCPort
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
