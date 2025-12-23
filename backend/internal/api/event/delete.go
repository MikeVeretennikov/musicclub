package event

import (
	"context"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *EventService) DeleteEvent(ctx context.Context, req *proto.EventId) (*emptypb.Empty, error) {
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
	if !helpers.PermissionAllowsEventEdit(perms) {
		return nil, status.Error(codes.PermissionDenied, "no rights to delete events")
	}

	res, err := db.ExecContext(ctx, `DELETE FROM event WHERE id = $1`, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete event: %v", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	return &emptypb.Empty{}, nil
}
