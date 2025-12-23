package song

import (
	"context"
	"database/sql"
)

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
