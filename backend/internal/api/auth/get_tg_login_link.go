package auth

import (
	"context"
	"database/sql"
	"fmt"
	"musicclubbot/backend/internal/config"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AuthService) GetTgLoginLink(ctx context.Context, req *proto.User) (*proto.TgLoginLinkResponse, error) {
	// Get user ID from context (user must be authenticated)
	userIDStr, err := helpers.UserIDFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Check if user already has Telegram linked
	var existingTgID sql.NullInt64
	err = db.QueryRowContext(ctx, `
		SELECT tg_user_id FROM app_user WHERE id = $1`,
		userID,
	).Scan(&existingTgID)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "query user: %v", err)
	}

	if existingTgID.Valid {
		return nil, status.Error(codes.AlreadyExists, "Telegram already linked to this account")
	}

	// Store the login token in tg_auth_user table
	var authId uuid.UUID
	err = db.QueryRowContext(ctx, `
		INSERT INTO tg_auth_user (user_id, tg_user_id)
		VALUES ($1, NULL)
		RETURNING (id)`,
		userID,
	).Scan(&authId)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "store tg auth session: %v", err)
	}

	cfg := ctx.Value("cfg").(config.Config)
	loginLink := fmt.Sprintf("https://t.me/%s?start=auth_%s", cfg.BotUsername, authId)

	return &proto.TgLoginLinkResponse{
		LoginLink: loginLink,
	}, nil
}
