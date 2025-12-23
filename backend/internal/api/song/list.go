package song

import (
	"context"
	"database/sql"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SongService) ListSongs(ctx context.Context, req *proto.ListSongsRequest) (*proto.ListSongsResponse, error) {
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	currentUserID, _ := helpers.UserIDFromCtx(ctx) // best effort; anonymous users just see editable=false

	limit := int(req.GetPageSize())
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := 0
	if tok := req.GetPageToken(); tok != "" {
		if v, err := strconv.Atoi(tok); err == nil && v >= 0 {
			offset = v
		}
	}

	args := []any{}
	where := ""
	if q := req.GetQuery(); q != "" {
		where = "WHERE title ILIKE $1 OR artist ILIKE $1"
		args = append(args, "%"+q+"%")
	}

	query := `
		SELECT id, title, artist, description, link_kind, link_url, COALESCE(created_by, NULL)
		FROM song
	` + where + `
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(len(args)+1) + `
		OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list songs: %v", err)
	}
	defer rows.Close()

	perms, _ := helpers.LoadPermissions(ctx, db, currentUserID)

	var songs []*proto.Song
	for rows.Next() {
		var sng proto.Song
		var linkKind, linkURL string
		var creatorID sql.NullString
		if err := rows.Scan(&sng.Id, &sng.Title, &sng.Artist, &sng.Description, &linkKind, &linkURL, &creatorID); err != nil {
			return nil, status.Errorf(codes.Internal, "scan song: %v", err)
		}
		sng.Link = &proto.SongLink{Kind: helpers.MapSongLinkType(linkKind), Url: linkURL}
		roles, err := helpers.LoadSongRoles(ctx, db, sng.Id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load roles: %v", err)
		}
		sng.AvailableRoles = roles
		sng.EditableByMe = helpers.PermissionAllowsSongEdit(perms, creatorID, currentUserID)
		songs = append(songs, &sng)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "iterate songs: %v", err)
	}

	nextToken := ""
	if len(songs) == limit {
		nextToken = strconv.Itoa(offset + limit)
	}

	return &proto.ListSongsResponse{
		Songs:         songs,
		NextPageToken: nextToken,
	}, nil
}
