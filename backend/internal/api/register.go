package api

import (
	"google.golang.org/grpc"

	authpb "musicclubbot/backend/proto"
	eventpb "musicclubbot/backend/proto"
	songpb "musicclubbot/backend/proto"
)

// Register wires all service handlers to the gRPC server.
func Register(server *grpc.Server) {
	authpb.RegisterAuthServiceServer(server, &AuthService{})
	songpb.RegisterSongServiceServer(server, &SongService{})
	eventpb.RegisterEventServiceServer(server, &EventService{})
}
