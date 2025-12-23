package auth

import (
	"context"
	"musicclubbot/backend/internal/helpers"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Authentication middleware
func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if helpers.PublicMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	tokenString := authHeaders[0]
	if !strings.HasPrefix(tokenString, "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	claims, err := VerifyToken(ctx, tokenString)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	db, err := helpers.DbFromCtx(ctx)
	if err == nil {
		var exists bool
		userID, parseErr := uuid.Parse(claims.UserID)
		if parseErr == nil {
			err = db.QueryRowContext(ctx,
				`SELECT EXISTS(SELECT 1 FROM app_user WHERE id = $1)`,
				userID,
			).Scan(&exists)

			if err == nil && !exists {
				return nil, status.Error(codes.Unauthenticated, "user no longer exists")
			}
		}
	}

	ctx = context.WithValue(ctx, "user_claims", claims)
	ctx = context.WithValue(ctx, "user_id", claims.UserID)

	return handler(ctx, req)
}
