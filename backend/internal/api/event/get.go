package event

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *EventService) GetEvent(ctx context.Context, req *proto.EventId) (*proto.EventDetails, error) {
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	currentUserID, _ := helpers.UserIDFromCtx(ctx)
	details, err := helpers.LoadEventDetails(ctx, db, req.GetId(), currentUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		return nil, status.Errorf(codes.Internal, "get event: %v", err)
	}
	return details, nil
}
