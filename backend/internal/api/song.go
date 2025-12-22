package api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	songpb "musicclubbot/backend/proto"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// SongService implements song catalog endpoints.
type SongService struct {
	songpb.UnimplementedSongServiceServer
}

func (s *SongService) ListSongs(ctx context.Context, req *songpb.ListSongsRequest) (*songpb.ListSongsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ListSongs not yet implemented")
}

func (s *SongService) GetSong(ctx context.Context, req *songpb.SongId) (*songpb.SongDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetSong not yet implemented")
}

func (s *SongService) CreateSong(ctx context.Context, req *songpb.CreateSongRequest) (*songpb.SongDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "CreateSong not yet implemented")
}

func (s *SongService) UpdateSong(ctx context.Context, req *songpb.UpdateSongRequest) (*songpb.SongDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "UpdateSong not yet implemented")
}

func (s *SongService) DeleteSong(ctx context.Context, req *songpb.SongId) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "DeleteSong not yet implemented")
}

func (s *SongService) JoinRole(ctx context.Context, req *songpb.JoinRoleRequest) (*songpb.SongDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "JoinRole not yet implemented")
}

func (s *SongService) LeaveRole(ctx context.Context, req *songpb.LeaveRoleRequest) (*songpb.SongDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "LeaveRole not yet implemented")
}
