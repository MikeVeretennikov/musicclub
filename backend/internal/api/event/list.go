package event

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *EventService) ListEvents(ctx context.Context, req *proto.ListEventsRequest) (*proto.ListEventsResponse, error) {
	db, err := helpers.DbFromCtx(ctx)
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

	var events []*proto.Event
	for rows.Next() {
		var ev proto.Event
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

	return &proto.ListEventsResponse{Events: events}, nil
}
