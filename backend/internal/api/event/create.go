package event

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *EventService) CreateEvent(ctx context.Context, req *proto.CreateEventRequest) (*proto.EventDetails, error) {
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
		return nil, status.Error(codes.PermissionDenied, "no rights to create events")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	var eventID string
	var startAt sql.NullTime
	if ts := req.GetStartAt(); ts != nil {
		startAt = sql.NullTime{Valid: true, Time: ts.AsTime()}
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO event (title, start_at, location, notify_day_before, notify_hour_before, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.GetTitle(), startAt, nullIfEmpty(req.GetLocation()), req.GetNotifyDayBefore(), req.GetNotifyHourBefore(), userID).Scan(&eventID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert event: %v", err)
	}

	if err := helpers.ReplaceTracklist(ctx, tx, eventID, req.GetTracklist()); err != nil {
		return nil, status.Errorf(codes.Internal, "set tracklist: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return helpers.LoadEventDetails(ctx, db, eventID, userID)
}
