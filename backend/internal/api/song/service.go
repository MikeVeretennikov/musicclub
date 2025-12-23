package song

import (
	"musicclubbot/backend/proto"
)

// SongService implements song catalog endpoints.
type SongService struct {
	proto.UnimplementedSongServiceServer
}
