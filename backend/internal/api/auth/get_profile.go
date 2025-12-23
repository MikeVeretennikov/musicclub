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
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *AuthService) GetProfile(ctx context.Context, req *emptypb.Empty) (*proto.ProfileResponse, error) {
	// Extract user ID from context
	userIDStr, err := helpers.UserIDFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid user ID format")
	}

	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get user profile
	var username, displayName string
	var avatarUrl sql.NullString
	var tgUserID sql.NullInt64
	var isChatMember bool
	var createdAt time.Time

	err = db.QueryRowContext(ctx, `
		SELECT username, display_name, avatar_url, tg_user_id, is_chat_member, created_at
		FROM app_user 
		WHERE id = $1`,
		userID,
	).Scan(&username, &displayName, &avatarUrl, &tgUserID, &isChatMember, &createdAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "query user: %v", err)
	}

	// Get user permissions
	permissions, err := helpers.GetUserPermissions(ctx, db, userID)
	if err != nil {
		// Use default permissions if we can't fetch
		permissions = &proto.PermissionSet{}
	}

	profile := &proto.User{
		Id:          userID.String(),
		Username:    username,
		DisplayName: displayName,
	}
	if avatarUrl.Valid {
		profile.AvatarUrl = avatarUrl.String
	}
	if tgUserID.Valid {
		profile.TelegramId = uint64(tgUserID.Int64)
	}

	return &proto.ProfileResponse{
		Profile:     profile,
		Permissions: permissions,
	}, nil
}
