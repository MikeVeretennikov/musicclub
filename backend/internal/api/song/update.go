package song

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SongService) UpdateSong(ctx context.Context, req *proto.UpdateSongRequest) (*proto.SongDetails, error) {
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
		return nil, status.Error(codes.PermissionDenied, "no rights to edit song")
	}

	linkKind, err := helpers.MapSongLinkKindToDB(req.GetLink().GetKind())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE song
		SET title = $1, artist = $2, description = $3, link_kind = $4, link_url = $5, updated_at = NOW()
		WHERE id = $6
	`, req.GetTitle(), req.GetArtist(), req.GetDescription(), linkKind, req.GetLink().GetUrl(), req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "update song: %v", err)
	}

	if err := replaceSongRoles(ctx, tx, req.GetId(), req.GetAvailableRoles()); err != nil {
		return nil, status.Errorf(codes.Internal, "set roles: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return helpers.LoadSongDetails(ctx, db, req.GetId(), userID)
}
