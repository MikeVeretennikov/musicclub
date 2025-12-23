package event

import "database/sql"

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return sql.NullString{}
	}
	return s
}
