package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"musicclubbot/backend/proto"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func DbFromCtx(ctx context.Context) (*sql.DB, error) {
	db, ok := ctx.Value("db").(*sql.DB)
	if !ok || db == nil {
		return nil, status.Error(codes.Internal, "database connection not available in context")
	}
	return db, nil
}

func UserIDFromCtx(ctx context.Context) (string, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return "", status.Error(codes.Unauthenticated, "user not authenticated")
	}
	return userID, nil
}

func LoadUserById(ctx context.Context, db *sql.DB, userID string) (*proto.User, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, display_name, username, COALESCE(avatar_url, '')
		FROM app_user WHERE id = $1
	`, userID)
	var u proto.User
	if err := row.Scan(&u.Id, &u.DisplayName, &u.Username, &u.AvatarUrl); err != nil {
		return nil, err
	}
	return &u, nil
}

func LoadUserByUsername(ctx context.Context, db *sql.DB, username string) (*proto.User, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, display_name, username, COALESCE(avatar_url, '')
		FROM app_user WHERE username = $1
	`, username)
	var u proto.User
	if err := row.Scan(&u.Id, &u.DisplayName, &u.Username, &u.AvatarUrl); err != nil {
		return nil, err
	}
	return &u, nil
}

func LoadPermissions(ctx context.Context, db *sql.DB, userID string) (*proto.PermissionSet, error) {
	row := db.QueryRowContext(ctx, `
		SELECT edit_own_participation, edit_any_participation,
		       edit_own_songs, edit_any_songs,
		       edit_events, edit_tracklists
		FROM user_permissions WHERE user_id = $1
	`, userID)
	var p proto.PermissionSet
	var joinOwn, joinAny, songsOwn, songsAny, events, tracks bool
	switch err := row.Scan(&joinOwn, &joinAny, &songsOwn, &songsAny, &events, &tracks); err {
	case nil:
		// ok
	case sql.ErrNoRows:
		// default permissions are all false
	default:
		return nil, err
	}

	p.Join = &proto.JoinPermissions{
		EditOwnParticipation: joinOwn,
		EditAnyParticipation: joinAny,
	}
	p.Songs = &proto.SongPermissions{
		EditOwnSongs: songsOwn,
		EditAnySongs: songsAny,
	}
	p.Events = &proto.EventPermissions{
		EditEvents:     events,
		EditTracklists: tracks,
	}
	return &p, nil
}

func MapSongLinkType(dbValue string) proto.SongLinkType {
	switch strings.ToLower(dbValue) {
	case "youtube":
		return proto.SongLinkType_SONG_LINK_TYPE_YOUTUBE
	case "yandex_music":
		return proto.SongLinkType_SONG_LINK_TYPE_YANDEX_MUSIC
	case "soundcloud":
		return proto.SongLinkType_SONG_LINK_TYPE_SOUNDCLOUD
	default:
		return proto.SongLinkType_SONG_LINK_TYPE_UNKNOWN
	}
}

func MapSongLinkKindToDB(kind proto.SongLinkType) (string, error) {
	switch kind {
	case proto.SongLinkType_SONG_LINK_TYPE_YOUTUBE:
		return "youtube", nil
	case proto.SongLinkType_SONG_LINK_TYPE_YANDEX_MUSIC:
		return "yandex_music", nil
	case proto.SongLinkType_SONG_LINK_TYPE_SOUNDCLOUD:
		return "soundcloud", nil
	default:
		return "", errors.New("unsupported song link type")
	}
}

func PermissionAllowsSongEdit(perms *proto.PermissionSet, ownerID sql.NullString, currentID string) bool {
	if perms == nil || perms.Songs == nil {
		return false
	}
	if perms.Songs.EditAnySongs {
		return true
	}
	return perms.Songs.EditOwnSongs && ownerID.String != "" && ownerID.String == currentID
}

func PermissionAllowsJoinEdit(perms *proto.PermissionSet, ownerID, currentID string) bool {
	if perms == nil || perms.Join == nil {
		return false
	}
	if perms.Join.EditAnyParticipation {
		return true
	}
	return perms.Join.EditOwnParticipation && ownerID != "" && ownerID == currentID
}

func PermissionAllowsEventEdit(perms *proto.PermissionSet) bool {
	return perms != nil && perms.Events != nil && perms.Events.EditEvents
}

func PermissionAllowsTracklistEdit(perms *proto.PermissionSet) bool {
	return perms != nil && perms.Events != nil && (perms.Events.EditTracklists || perms.Events.EditEvents)
}

func LoadSongDetails(ctx context.Context, db *sql.DB, songID, currentUserID string) (*proto.SongDetails, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, title, artist, description, link_kind, link_url, COALESCE(created_by, NULL)
		FROM song WHERE id = $1
	`, songID)
	var s proto.Song
	var linkKind, linkURL string
	var creatorID sql.NullString
	if err := row.Scan(&s.Id, &s.Title, &s.Artist, &s.Description, &linkKind, &linkURL, &creatorID); err != nil {
		return nil, err
	}
	s.Link = &proto.SongLink{Kind: MapSongLinkType(linkKind), Url: linkURL}

	roles, err := LoadSongRoles(ctx, db, songID)
	if err != nil {
		return nil, err
	}
	s.AvailableRoles = roles

	perms, err := LoadPermissions(ctx, db, currentUserID)
	if err != nil {
		return nil, err
	}
	s.EditableByMe = PermissionAllowsSongEdit(perms, creatorID, currentUserID)

	assignments, err := LoadSongAssignments(ctx, db, songID)
	if err != nil {
		return nil, err
	}

	return &proto.SongDetails{
		Song:        &s,
		Assignments: assignments,
		Permissions: perms,
	}, nil
}

func LoadSongRoles(ctx context.Context, db *sql.DB, songID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT role FROM song_role WHERE song_id = $1 ORDER BY role`, songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

func LoadSongAssignments(ctx context.Context, db *sql.DB, songID string) ([]*proto.RoleAssignment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT sra.role,
		       au.id, au.display_name, COALESCE(au.username, ''), COALESCE(au.avatar_url, ''),
		       sra.joined_at
		FROM song_role_assignment sra
		JOIN app_user au ON sra.user_id = au.id
		WHERE sra.song_id = $1
		ORDER BY sra.joined_at ASC
	`, songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*proto.RoleAssignment
	for rows.Next() {
		var role, uid, display, username, avatar string
		var joined time.Time
		if err := rows.Scan(&role, &uid, &display, &username, &avatar, &joined); err != nil {
			return nil, err
		}
		items = append(items, &proto.RoleAssignment{
			Role: role,
			User: &proto.User{
				Id:          uid,
				DisplayName: display,
				Username:    username,
				AvatarUrl:   avatar,
			},
			JoinedAt: timestamppb.New(joined),
		})
	}
	return items, rows.Err()
}

func LoadEventDetails(ctx context.Context, db *sql.DB, eventID, currentUserID string) (*proto.EventDetails, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, title, start_at, location, notify_day_before, notify_hour_before
		FROM event WHERE id = $1
	`, eventID)
	var e proto.Event
	var start sql.NullTime
	if err := row.Scan(&e.Id, &e.Title, &start, &e.Location, &e.NotifyDayBefore, &e.NotifyHourBefore); err != nil {
		return nil, err
	}
	if start.Valid {
		e.StartAt = timestamppb.New(start.Time)
	}

	tracklist, err := LoadTracklist(ctx, db, eventID)
	if err != nil {
		return nil, err
	}

	participants, err := LoadEventParticipants(ctx, db, eventID)
	if err != nil {
		return nil, err
	}

	perms, err := LoadPermissions(ctx, db, currentUserID)
	if err != nil {
		return nil, err
	}

	return &proto.EventDetails{
		Event:        &e,
		Tracklist:    tracklist,
		Participants: participants,
		Permissions:  perms,
	}, nil
}

func LoadTracklist(ctx context.Context, db *sql.DB, eventID string) (*proto.Tracklist, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT position, COALESCE(song_id, ''), COALESCE(custom_title, ''), COALESCE(custom_artist, '')
		FROM event_track_item
		WHERE event_id = $1
		ORDER BY position
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*proto.TrackItem
	for rows.Next() {
		var pos int32
		var songID, customTitle, customArtist string
		if err := rows.Scan(&pos, &songID, &customTitle, &customArtist); err != nil {
			return nil, err
		}
		items = append(items, &proto.TrackItem{
			Order:        uint32(pos),
			SongId:       songID,
			CustomTitle:  customTitle,
			CustomArtist: customArtist,
		})
	}
	return &proto.Tracklist{Items: items}, rows.Err()
}

func LoadEventParticipants(ctx context.Context, db *sql.DB, eventID string) ([]*proto.RoleAssignment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ep.role,
		       au.id, au.display_name, COALESCE(au.username, ''), COALESCE(au.avatar_url, ''),
		       ep.joined_at
		FROM event_participant ep
		JOIN app_user au ON ep.user_id = au.id
		WHERE ep.event_id = $1
		ORDER BY ep.joined_at
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*proto.RoleAssignment
	for rows.Next() {
		var role, uid, display, username, avatar string
		var joined time.Time
		if err := rows.Scan(&role, &uid, &display, &username, &avatar, &joined); err != nil {
			return nil, err
		}
		items = append(items, &proto.RoleAssignment{
			Role: role,
			User: &proto.User{
				Id:          uid,
				DisplayName: display,
				Username:    username,
				AvatarUrl:   avatar,
			},
			JoinedAt: timestamppb.New(joined),
		})
	}
	return items, rows.Err()
}

func ReplaceTracklist(ctx context.Context, tx *sql.Tx, eventID string, tracklist *proto.Tracklist) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM event_track_item WHERE event_id = $1`, eventID); err != nil {
		return err
	}
	if tracklist == nil {
		return nil
	}
	for _, item := range tracklist.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_track_item (event_id, position, song_id, custom_title, custom_artist)
			VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''))
		`, eventID, item.GetOrder(), item.GetSongId(), item.GetCustomTitle(), item.GetCustomArtist()); err != nil {
			return err
		}
	}
	return nil
}

// Helper functions
func AcceptablePassword(password string) bool {
	if password == "" {
		return false
	}
	if len(password) < 8 {
		return false
	}
	// Add more complexity checks if needed
	// e.g., require at least one uppercase, one lowercase, one number, one special char
	return true
}

func GetUserPermissions(
	ctx context.Context,
	db any,
	userID uuid.UUID,
) (*proto.PermissionSet, error) {

	type queryRower interface {
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}

	var q queryRower

	switch d := db.(type) {
	case *sql.DB:
		q = d
	case *sql.Tx:
		q = d
	default:
		return nil, fmt.Errorf("unsupported db type %T", db)
	}

	permissions := &proto.PermissionSet{
		Join:   &proto.JoinPermissions{},
		Songs:  &proto.SongPermissions{},
		Events: &proto.EventPermissions{},
	}

	err := q.QueryRowContext(ctx, `
		SELECT
			edit_own_participation,
			edit_any_participation,
			edit_own_songs,
			edit_any_songs,
			edit_events,
			edit_tracklists
		FROM user_permissions
		WHERE user_id = $1
	`, userID).Scan(
		&permissions.Join.EditOwnParticipation,
		&permissions.Join.EditAnyParticipation,
		&permissions.Songs.EditOwnSongs,
		&permissions.Songs.EditAnySongs,
		&permissions.Events.EditEvents,
		&permissions.Events.EditTracklists,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// user has no explicit permissions → return defaults
			return permissions, nil
		}
		return nil, err
	}

	return permissions, nil
}

var PublicMethods = map[string]bool{
	"/musicclub.auth.AuthService/Login":              true,
	"/musicclub.auth.AuthService/Register":           true,
	"/musicclub.auth.AuthService/Refresh":            true,
	"/musicclub.auth.AuthService/TelegramWebAppAuth": true,
}
