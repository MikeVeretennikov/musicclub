package event

import (
	"musicclubbot/backend/proto"
)

// EventService implements event and tracklist endpoints.
type EventService struct {
	proto.UnimplementedEventServiceServer
}
