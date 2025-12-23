package auth

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AuthService) Refresh(ctx context.Context, req *proto.RefreshRequest) (*proto.TokenPair, error) {
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	refreshToken := req.GetRefreshToken()
	if refreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	// Verify refresh token exists and is valid
	var userID uuid.UUID
	var expiresAt time.Time

	err = db.QueryRowContext(ctx, `
		SELECT user_id, expires_at 
		FROM refresh_tokens 
		WHERE token = $1 AND expires_at > NOW()`,
		refreshToken,
	).Scan(&userID, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
		}
		return nil, status.Errorf(codes.Internal, "query refresh token: %v", err)
	}

	// Get user info for new token
	var username string
	err = db.QueryRowContext(ctx, `
		SELECT username FROM app_user WHERE id = $1`,
		userID,
	).Scan(&username)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "query user: %v", err)
	}

	// Generate new tokens
	newAccessToken, err := GenerateAccessToken(ctx, userID, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate access token: %v", err)
	}

	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate refresh token: %v", err)
	}

	// Update refresh token in database
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	// Delete old refresh token
	_, err = tx.ExecContext(ctx, `
		DELETE FROM refresh_tokens WHERE token = $1`,
		refreshToken)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete old token: %v", err)
	}

	// Store new refresh token
	newRefreshExpiresAt := time.Now().Add(RefreshTokenExp)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES (gen_random_uuid(), $1, $2, $3)`,
		userID, newRefreshToken, newRefreshExpiresAt)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "store new token: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return &proto.TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
