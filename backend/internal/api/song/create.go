package song

import (
	"context"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SongService) CreateSong(ctx context.Context, req *proto.CreateSongRequest) (*proto.SongDetails, error) {
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
	if perms.Songs == nil || (!perms.Songs.EditOwnSongs && !perms.Songs.EditAnySongs) {
		return nil, status.Error(codes.PermissionDenied, "no rights to create songs")
	}

	linkKind, err := helpers.MapSongLinkKindToDB(req.GetLink().GetKind())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var songID string
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO song (title, artist, description, link_kind, link_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.GetTitle(), req.GetArtist(), req.GetDescription(), linkKind, req.GetLink().GetUrl(), userID).Scan(&songID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert song: %v", err)
	}

	if err := replaceSongRoles(ctx, tx, songID, req.GetAvailableRoles()); err != nil {
		return nil, status.Errorf(codes.Internal, "set roles: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return helpers.LoadSongDetails(ctx, db, songID, userID)
}
