package auth

import (
	"context"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *AuthService) Register(ctx context.Context, req *proto.RegisterUserRequest) (*proto.AuthSession, error) {
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	username := req.GetCredentials().GetUsername()
	if username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	var exists bool
	err = db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM app_user WHERE username = $1)`,
		username,
	).Scan(&exists)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "check existing username: %v", err)
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "username already taken")
	}

	password := req.GetCredentials().GetPassword()
	if !helpers.AcceptablePassword(password) {
		return nil, status.Error(codes.InvalidArgument, "password does not meet complexity requirements")
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "hash password: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	var userID uuid.UUID
	var displayName string
	var avatarUrl *string

	profile := req.GetProfile()
	if profile != nil {
		displayName = profile.GetDisplayName()
		if profile.GetAvatarUrl() != "" {
			avatarUrl = &profile.AvatarUrl
		}
	}

	// Use default display name if not provided
	if displayName == "" {
		displayName = username
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO app_user (username, password_hash, display_name, avatar_url, is_chat_member) 
		VALUES ($1, $2, $3, $4, FALSE)
		RETURNING id, display_name, avatar_url`,
		username,
		hashedPassword,
		displayName,
		avatarUrl,
	).Scan(&userID, &displayName, &avatarUrl)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert user: %v", err)
	}

	// челику без тг запрещено все
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_permissions (user_id, edit_own_participation, edit_any_participation, 
		                              edit_own_songs, edit_any_songs, edit_events, edit_tracklists)
		VALUES ($1, FALSE, FALSE, FALSE, FALSE, FALSE, FALSE)`,
		userID,
	)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "set default permissions: %v", err)
	}

	// Generate JWT tokens
	accessToken, err := GenerateAccessToken(ctx, userID, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate access token: %v", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate refresh token: %v", err)
	}

	// Store refresh token in database
	refreshExpiresAt := time.Now().Add(RefreshTokenExp)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES (gen_random_uuid(), $1, $2, $3)`,
		userID, refreshToken, refreshExpiresAt)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "store refresh token: %v", err)
	}

	// Get permissions for response
	permissions, err := helpers.GetUserPermissions(ctx, tx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get user permissions: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	// Create user profile response
	profileResp := &proto.User{
		Id:          userID.String(),
		Username:    username,
		DisplayName: displayName,
	}
	if avatarUrl != nil {
		profileResp.AvatarUrl = *avatarUrl
	}

	// Check if user is chat member
	var isChatMember bool
	err = db.QueryRowContext(ctx,
		`SELECT is_chat_member FROM app_user WHERE id = $1`,
		userID,
	).Scan(&isChatMember)

	if err != nil {
		isChatMember = false
	}

	return &proto.AuthSession{
		Tokens: &proto.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
		Iat:            uint64(time.Now().Unix()),
		Exp:            uint64(time.Now().Add(AccessTokenExp).Unix()),
		IsChatMember:   isChatMember,
		JoinRequestUrl: "https://t.me/your_musicclub_bot?start=join", // Replace with your bot
		Profile:        profileResp,
		Permissions:    permissions,
	}, nil
}
