package event

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *EventService) UpdateEvent(ctx context.Context, req *proto.UpdateEventRequest) (*proto.EventDetails, error) {
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
		return nil, status.Error(codes.PermissionDenied, "no rights to update events")
	}

	var startAt sql.NullTime
	if ts := req.GetStartAt(); ts != nil {
		startAt = sql.NullTime{Valid: true, Time: ts.AsTime()}
	}

	res, err := db.ExecContext(ctx, `
		UPDATE event
		SET title = $1, start_at = $2, location = $3, notify_day_before = $4, notify_hour_before = $5, updated_at = NOW()
		WHERE id = $6
	`, req.GetTitle(), startAt, nullIfEmpty(req.GetLocation()), req.GetNotifyDayBefore(), req.GetNotifyHourBefore(), req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update event: %v", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	return helpers.LoadEventDetails(ctx, db, req.GetId(), userID)
}
