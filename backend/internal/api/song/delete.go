package song

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *SongService) DeleteSong(ctx context.Context, req *proto.SongId) (*emptypb.Empty, error) {
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

	var creatorID sql.NullString
	row := db.QueryRowContext(ctx, `SELECT COALESCE(created_by, NULL) FROM song WHERE id = $1`, req.GetId())
	if err := row.Scan(&creatorID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "song not found")
		}
		return nil, status.Errorf(codes.Internal, "load song: %v", err)
	}
	if !helpers.PermissionAllowsSongEdit(perms, creatorID, userID) {
		return nil, status.Error(codes.PermissionDenied, "no rights to delete song")
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM song WHERE id = $1`, req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete song: %v", err)
	}
	return &emptypb.Empty{}, nil
}
