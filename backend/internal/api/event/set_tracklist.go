package event

import (
	"context"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *EventService) SetTracklist(ctx context.Context, req *proto.SetTracklistRequest) (*proto.EventDetails, error) {
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
	if !helpers.PermissionAllowsTracklistEdit(perms) {
		return nil, status.Error(codes.PermissionDenied, "no rights to edit tracklists")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	if err := helpers.ReplaceTracklist(ctx, tx, req.GetEventId(), req.GetTracklist()); err != nil {
		return nil, status.Errorf(codes.Internal, "set tracklist: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return helpers.LoadEventDetails(ctx, db, req.GetEventId(), userID)
}
