package api

import (
	"context"
	"database/sql"
	"strconv"

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
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	currentUserID, _ := userIDFromCtx(ctx) // best effort; anonymous users just see editable=false

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
		SELECT id, title, artist, description, link_kind, link_url, COALESCE(created_by, '')
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

	perms, _ := loadPermissions(ctx, db, currentUserID)

	var songs []*songpb.Song
	for rows.Next() {
		var sng songpb.Song
		var linkKind, linkURL, creatorID string
		if err := rows.Scan(&sng.Id, &sng.Title, &sng.Artist, &sng.Description, &linkKind, &linkURL, &creatorID); err != nil {
			return nil, status.Errorf(codes.Internal, "scan song: %v", err)
		}
		sng.Link = &songpb.SongLink{Kind: mapSongLinkType(linkKind), Url: linkURL}
		roles, err := loadSongRoles(ctx, db, sng.Id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load roles: %v", err)
		}
		sng.AvailableRoles = roles
		sng.EditableByMe = permissionAllowsSongEdit(perms, creatorID, currentUserID)
		songs = append(songs, &sng)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "iterate songs: %v", err)
	}

	nextToken := ""
	if len(songs) == limit {
		nextToken = strconv.Itoa(offset + limit)
	}

	return &songpb.ListSongsResponse{
		Songs:        songs,
		NextPageToken: nextToken,
	}, nil
}

func (s *SongService) GetSong(ctx context.Context, req *songpb.SongId) (*songpb.SongDetails, error) {
	db, err := dbFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	currentUserID, _ := userIDFromCtx(ctx)
	details, err := loadSongDetails(ctx, db, req.GetId(), currentUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "song not found")
		}
		return nil, status.Errorf(codes.Internal, "get song: %v", err)
	}
	return details, nil
}

func (s *SongService) CreateSong(ctx context.Context, req *songpb.CreateSongRequest) (*songpb.SongDetails, error) {
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
	if perms.Songs == nil || (!perms.Songs.EditOwnSongs && !perms.Songs.EditAnySongs) {
		return nil, status.Error(codes.PermissionDenied, "no rights to create songs")
	}

	linkKind, err := mapSongLinkKindToDB(req.GetLink().GetKind())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var songID string
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO song (title, artist, description, link_kind, link_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.GetTitle(), req.GetArtist(), req.GetDescription(), linkKind, req.GetLink().GetUrl(), userID).Scan(&songID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert song: %v", err)
	}

	if err := replaceSongRoles(ctx, tx, songID, req.GetAvailableRoles()); err != nil {
		return nil, status.Errorf(codes.Internal, "set roles: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return loadSongDetails(ctx, db, songID, userID)
}

func (s *SongService) UpdateSong(ctx context.Context, req *songpb.UpdateSongRequest) (*songpb.SongDetails, error) {
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

	var creatorID string
	row := db.QueryRowContext(ctx, `SELECT COALESCE(created_by, '') FROM song WHERE id = $1`, req.GetId())
	if err := row.Scan(&creatorID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "song not found")
		}
		return nil, status.Errorf(codes.Internal, "load song: %v", err)
	}
	if !permissionAllowsSongEdit(perms, creatorID, userID) {
		return nil, status.Error(codes.PermissionDenied, "no rights to edit song")
	}

	linkKind, err := mapSongLinkKindToDB(req.GetLink().GetKind())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE song
		SET title = $1, artist = $2, description = $3, link_kind = $4, link_url = $5, updated_at = NOW()
		WHERE id = $6
	`, req.GetTitle(), req.GetArtist(), req.GetDescription(), linkKind, req.GetLink().GetUrl(), req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "update song: %v", err)
	}

	if err := replaceSongRoles(ctx, tx, req.GetId(), req.GetAvailableRoles()); err != nil {
		return nil, status.Errorf(codes.Internal, "set roles: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	return loadSongDetails(ctx, db, req.GetId(), userID)
}

func (s *SongService) DeleteSong(ctx context.Context, req *songpb.SongId) (*emptypb.Empty, error) {
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

	var creatorID string
	row := db.QueryRowContext(ctx, `SELECT COALESCE(created_by, '') FROM song WHERE id = $1`, req.GetId())
	if err := row.Scan(&creatorID); err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "song not found")
		}
		return nil, status.Errorf(codes.Internal, "load song: %v", err)
	}
	if !permissionAllowsSongEdit(perms, creatorID, userID) {
		return nil, status.Error(codes.PermissionDenied, "no rights to delete song")
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM song WHERE id = $1`, req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "delete song: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *SongService) JoinRole(ctx context.Context, req *songpb.JoinRoleRequest) (*songpb.SongDetails, error) {
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
	if !permissionAllowsJoinEdit(perms, userID, userID) {
		return nil, status.Error(codes.PermissionDenied, "no rights to join roles")
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO song_role_assignment (song_id, role, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (song_id, role, user_id) DO NOTHING
	`, req.GetSongId(), req.GetRole(), userID); err != nil {
		return nil, status.Errorf(codes.Internal, "join role: %v", err)
	}

	return loadSongDetails(ctx, db, req.GetSongId(), userID)
}

func (s *SongService) LeaveRole(ctx context.Context, req *songpb.LeaveRoleRequest) (*songpb.SongDetails, error) {
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
	if !permissionAllowsJoinEdit(perms, userID, userID) {
		return nil, status.Error(codes.PermissionDenied, "no rights to leave roles")
	}

	if _, err := db.ExecContext(ctx, `
		DELETE FROM song_role_assignment WHERE song_id = $1 AND role = $2 AND user_id = $3
	`, req.GetSongId(), req.GetRole(), userID); err != nil {
		return nil, status.Errorf(codes.Internal, "leave role: %v", err)
	}

	return loadSongDetails(ctx, db, req.GetSongId(), userID)
}

func replaceSongRoles(ctx context.Context, tx *sql.Tx, songID string, roles []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM song_role WHERE song_id = $1`, songID); err != nil {
		return err
	}
	for _, r := range roles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO song_role (song_id, role) VALUES ($1, $2)`, songID, r); err != nil {
			return err
		}
	}
	return nil
}
