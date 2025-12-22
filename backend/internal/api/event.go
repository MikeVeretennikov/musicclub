package api

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventpb "musicclubbot/backend/proto"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// EventService implements event and tracklist endpoints.
type EventService struct {
	eventpb.UnimplementedEventServiceServer
}

func (s *EventService) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	args := []any{}
	clauses := []string{}
	if req.GetFrom() != nil {
		clauses = append(clauses, "start_at >= $"+strconv.Itoa(len(args)+1))
		args = append(args, time.Unix(req.GetFrom().Seconds, int64(req.GetFrom().Nanos)))
	}
	if req.GetTo() != nil {
		clauses = append(clauses, "start_at <= $"+strconv.Itoa(len(args)+1))
		args = append(args, time.Unix(req.GetTo().Seconds, int64(req.GetTo().Nanos)))
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}

	limit := req.GetLimit()
	if limit == 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, `
		SELECT id, title, start_at, location, notify_day_before, notify_hour_before
		FROM event
	`+where+`
		ORDER BY start_at NULLS LAST
		LIMIT $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list events: %v", err)
	}
	defer rows.Close()

	var events []*eventpb.Event
	for rows.Next() {
		var ev eventpb.Event
		var start sql.NullTime
		if err := rows.Scan(&ev.Id, &ev.Title, &start, &ev.Location, &ev.NotifyDayBefore, &ev.NotifyHourBefore); err != nil {
			return nil, status.Errorf(codes.Internal, "scan event: %v", err)
		}
		if start.Valid {
			ev.StartAt = timestamppb.New(start.Time)
		}
		events = append(events, &ev)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "iterate events: %v", err)
	}

	return &eventpb.ListEventsResponse{Events: events}, nil
}

func (s *EventService) GetEvent(ctx context.Context, req *eventpb.EventId) (*eventpb.EventDetails, error) {
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	currentUserID, _ := userIDFromCtx(ctx)
	details, err := loadEventDetails(ctx, db, req.GetId(), currentUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		return nil, status.Errorf(codes.Internal, "get event: %v", err)
	}
	return details, nil
}

func (s *EventService) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.EventDetails, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	perms, err := loadPermissions(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load permissions: %v", err)
	}
	if !permissionAllowsEventEdit(perms) {
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

	if err := replaceTracklist(ctx, tx, eventID, req.GetTracklist()); err != nil {
		return nil, status.Errorf(codes.Internal, "set tracklist: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return loadEventDetails(ctx, db, eventID, userID)
}

func (s *EventService) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.EventDetails, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	perms, err := loadPermissions(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load permissions: %v", err)
	}
	if !permissionAllowsEventEdit(perms) {
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
	return loadEventDetails(ctx, db, req.GetId(), userID)
}

func (s *EventService) DeleteEvent(ctx context.Context, req *eventpb.EventId) (*emptypb.Empty, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	perms, err := loadPermissions(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load permissions: %v", err)
	}
	if !permissionAllowsEventEdit(perms) {
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

func (s *EventService) SetTracklist(ctx context.Context, req *eventpb.SetTracklistRequest) (*eventpb.EventDetails, error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	perms, err := loadPermissions(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load permissions: %v", err)
	}
	if !permissionAllowsTracklistEdit(perms) {
		return nil, status.Error(codes.PermissionDenied, "no rights to edit tracklists")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	if err := replaceTracklist(ctx, tx, req.GetEventId(), req.GetTracklist()); err != nil {
		return nil, status.Errorf(codes.Internal, "set tracklist: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return loadEventDetails(ctx, db, req.GetEventId(), userID)
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return sql.NullString{}
	}
	return s
}
