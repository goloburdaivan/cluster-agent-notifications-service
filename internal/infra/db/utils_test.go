package db

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUIDToString_Valid(t *testing.T) {
	u := pgtype.UUID{
		Bytes: [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
		Valid: true,
	}
	result := UUIDToString(u)
	assert.Equal(t, "12345678-9abc-def0-1234-56789abcdef0", result)
}

func TestUUIDToString_Invalid(t *testing.T) {
	u := pgtype.UUID{Valid: false}
	result := UUIDToString(u)
	assert.Empty(t, result)
}

func TestStringToUUID_Valid(t *testing.T) {
	u, err := StringToUUID("12345678-9abc-def0-1234-56789abcdef0")
	require.NoError(t, err)
	assert.True(t, u.Valid)
}

func TestStringToUUID_Invalid(t *testing.T) {
	_, err := StringToUUID("not-a-uuid")
	assert.Error(t, err)
}

func TestStringToUUID_Roundtrip(t *testing.T) {
	original := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	u, err := StringToUUID(original)
	require.NoError(t, err)

	result := UUIDToString(u)
	assert.Equal(t, original, result)
}
