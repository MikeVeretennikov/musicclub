package auth

import (
	"musicclubbot/backend/proto"
)

// AuthService implements auth-related gRPC endpoints.
type AuthService struct {
	proto.UnimplementedAuthServiceServer
	// You might want to add dependencies like a Telegram bot client here
	// telegramBot *tgbotapi.BotAPI
}
