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

func (s *AuthService) Login(ctx context.Context, req *proto.Credentials) (*proto.AuthSession, error) {
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	username := req.GetUsername()
	password := req.GetPassword()

	if username == "" || password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	// Get user from database
	var userID uuid.UUID
	var hashedPassword string
	var displayName string
	var avatarUrl sql.NullString
	var isChatMember bool
	var createdAt time.Time

	err = db.QueryRowContext(ctx, `
		SELECT id, password_hash, display_name, avatar_url, is_chat_member, created_at
		FROM app_user 
		WHERE username = $1`,
		username,
	).Scan(&userID, &hashedPassword, &displayName, &avatarUrl, &isChatMember, &createdAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "query user: %v", err)
	}

	// Verify password
	if !CheckPasswordHash(password, hashedPassword) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Generate new tokens
	accessToken, err := GenerateAccessToken(ctx, userID, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate access token: %v", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate refresh token: %v", err)
	}

	// Store refresh token and invalidate old ones
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	// Invalidate old refresh tokens for this user
	_, err = tx.ExecContext(ctx, `
			DELETE FROM refresh_tokens 
			WHERE user_id = $1`,
		userID)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalidate old tokens: %v", err)
	}

	// Store new refresh token
	refreshExpiresAt := time.Now().Add(RefreshTokenExp)
	_, err = tx.ExecContext(ctx, `
			INSERT INTO refresh_tokens (id, user_id, token, expires_at)
			VALUES (gen_random_uuid(), $1, $2, $3)`,
		userID, refreshToken, refreshExpiresAt)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "store refresh token: %v", err)
	}

	// Get user permissions
	permissions, err := helpers.GetUserPermissions(ctx, tx, userID)
	if err != nil {
		// Use default permissions if we can't fetch
		permissions = &proto.PermissionSet{}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	// Create user profile
	profile := &proto.User{
		Id:          userID.String(),
		Username:    username,
		DisplayName: displayName,
	}
	if avatarUrl.Valid {
		profile.AvatarUrl = avatarUrl.String
	}

	return &proto.AuthSession{
		Tokens: &proto.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
		Iat:            uint64(time.Now().Unix()),
		Exp:            uint64(time.Now().Add(AccessTokenExp).Unix()),
		IsChatMember:   isChatMember,
		JoinRequestUrl: "https://t.me/your_musicclub_bot?start=join", // TODO start link generation
		Profile:        profile,
		Permissions:    permissions,
	}, nil
}
