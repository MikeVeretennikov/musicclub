package api

import (
	"musicclubbot/backend/internal/api/auth"
	"musicclubbot/backend/internal/api/event"
	"musicclubbot/backend/internal/api/song"

	"google.golang.org/grpc"

	authpb "musicclubbot/backend/proto"
	eventpb "musicclubbot/backend/proto"
	songpb "musicclubbot/backend/proto"
)

// Register wires all service handlers to the gRPC server.
func Register(server *grpc.Server) {
	authpb.RegisterAuthServiceServer(server, &auth.AuthService{})
	songpb.RegisterSongServiceServer(server, &song.SongService{})
	eventpb.RegisterEventServiceServer(server, &event.EventService{})
}
