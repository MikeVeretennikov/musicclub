package song

import (
	"context"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SongService) JoinRole(ctx context.Context, req *proto.JoinRoleRequest) (*proto.SongDetails, error) {
	userID, err := helpers.UserIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	perms, err := helpers.LoadPermissions(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load permissions: %v", err)
	}
	if !helpers.PermissionAllowsJoinEdit(perms, userID, userID) {
		return nil, status.Error(codes.PermissionDenied, "no rights to join roles")
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO song_role_assignment (song_id, role, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (song_id, role, user_id) DO NOTHING
	`, req.GetSongId(), req.GetRole(), userID); err != nil {
		return nil, status.Errorf(codes.Internal, "join role: %v", err)
	}

	return helpers.LoadSongDetails(ctx, db, req.GetSongId(), userID)
}
