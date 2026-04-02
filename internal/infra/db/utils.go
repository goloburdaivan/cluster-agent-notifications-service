package db

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func UUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}

	return uuid.UUID(u.Bytes).String()
}

func StringToUUID(id string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(id)
	return u, err
}
