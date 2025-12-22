package api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	eventpb "musicclubbot/backend/proto"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// EventService implements event and tracklist endpoints.
type EventService struct {
	eventpb.UnimplementedEventServiceServer
}

func (s *EventService) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ListEvents not yet implemented")
}

func (s *EventService) GetEvent(ctx context.Context, req *eventpb.EventId) (*eventpb.EventDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetEvent not yet implemented")
}

func (s *EventService) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.EventDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "CreateEvent not yet implemented")
}

func (s *EventService) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.EventDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "UpdateEvent not yet implemented")
}

func (s *EventService) DeleteEvent(ctx context.Context, req *eventpb.EventId) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "DeleteEvent not yet implemented")
}

func (s *EventService) SetTracklist(ctx context.Context, req *eventpb.SetTracklistRequest) (*eventpb.EventDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "SetTracklist not yet implemented")
}
