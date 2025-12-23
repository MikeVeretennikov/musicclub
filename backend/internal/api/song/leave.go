package song

import (
	"context"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SongService) LeaveRole(ctx context.Context, req *proto.LeaveRoleRequest) (*proto.SongDetails, error) {
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
		return nil, status.Error(codes.PermissionDenied, "no rights to leave roles")
	}

	if _, err := db.ExecContext(ctx, `
		DELETE FROM song_role_assignment WHERE song_id = $1 AND role = $2 AND user_id = $3
	`, req.GetSongId(), req.GetRole(), userID); err != nil {
		return nil, status.Errorf(codes.Internal, "leave role: %v", err)
	}

	return helpers.LoadSongDetails(ctx, db, req.GetSongId(), userID)
}
