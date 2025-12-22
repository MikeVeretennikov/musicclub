package api

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authpb "musicclubbot/backend/proto"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// AuthService implements auth-related gRPC endpoints.
type AuthService struct {
	authpb.UnimplementedAuthServiceServer
}

func (s *AuthService) LoginWithTelegram(ctx context.Context, req *authpb.TgLoginRequest) (*authpb.AuthSession, error) {
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetTgUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "tg_user_id is required")
	}

	userID, isMember, err := upsertUserByTelegram(ctx, db, req.GetTgUserId(), req.GetInitData())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "upsert user: %v", err)
	}

	profile, err := loadUser(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load user: %v", err)
	}

	perm, err := loadPermissions(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load permissions: %v", err)
	}

	joinURL := ""
	if !isMember {
		token, err := ensureJoinRequest(ctx, db, userID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ensure join request: %v", err)
		}
		base := os.Getenv("CHAT_LINK")
		if base == "" {
			base = "https://t.me/joinchat"
		}
		joinURL = fmt.Sprintf("%s?start=%s", base, token)
	}

	now := time.Now().UTC()
	exp := now.Add(time.Duration(defaultJWTTTLSeconds()) * time.Second)

	// For now we return a placeholder token so the frontend can proceed.
	token := fmt.Sprintf("user:%s", userID)

	return &authpb.AuthSession{
		AccessToken:    token,
		Iat:            uint64(now.Unix()),
		Exp:            uint64(exp.Unix()),
		IsChatMember:   isMember,
		JoinRequestUrl: joinURL,
		Profile:        profile,
		Permissions:    perm,
	}, nil
}

func (s *AuthService) GetProfile(ctx context.Context, _ *emptypb.Empty) (*authpb.ProfileResponse, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := loadUser(ctx, db, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "load user: %v", err)
	}

	perm, err := loadPermissions(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load permissions: %v", err)
	}

	return &authpb.ProfileResponse{
		Profile:     profile,
		Permissions: perm,
	}, nil
}
