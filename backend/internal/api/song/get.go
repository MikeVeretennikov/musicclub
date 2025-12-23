package song

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SongService) GetSong(ctx context.Context, req *proto.SongId) (*proto.SongDetails, error) {
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	currentUserID, _ := helpers.UserIDFromCtx(ctx)
	details, err := helpers.LoadSongDetails(ctx, db, req.GetId(), currentUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "song not found")
		}
		return nil, status.Errorf(codes.Internal, "get song: %v", err)
	}
	return details, nil
}
