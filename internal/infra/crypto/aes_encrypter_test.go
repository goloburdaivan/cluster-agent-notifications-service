package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestEncryptDecrypt(t *testing.T) {
	enc, err := NewAESEncrypter(newTestKey(t))
	require.NoError(t, err)

	plaintext := []byte("hello world, this is a secret message")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_ProducesDifferentCiphertexts(t *testing.T) {
	enc, err := NewAESEncrypter(newTestKey(t))
	require.NoError(t, err)

	plaintext := []byte("test data")
	ct1, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	ct2, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	enc, err := NewAESEncrypter(newTestKey(t))
	require.NoError(t, err)

	_, err = enc.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	enc, err := NewAESEncrypter(newTestKey(t))
	require.NoError(t, err)

	_, err = enc.Decrypt("YQ==")
	assert.Error(t, err)
}

func TestDecrypt_TamperedData(t *testing.T) {
	enc, err := NewAESEncrypter(newTestKey(t))
	require.NoError(t, err)

	ct, err := enc.Encrypt([]byte("original"))
	require.NoError(t, err)

	tampered := ct[:len(ct)-2] + "XX"
	_, err = enc.Decrypt(tampered)
	assert.Error(t, err)
}

func TestNewAESEncrypter_InvalidKeySize(t *testing.T) {
	_, err := NewAESEncrypter([]byte("short"))
	assert.Error(t, err)
}

func TestNewAESEncrypter_ValidKeySizes(t *testing.T) {
	for _, size := range []int{16, 24, 32} {
		key := make([]byte, size)
		_, err := rand.Read(key)
		require.NoError(t, err)

		enc, err := NewAESEncrypter(key)
		assert.NoError(t, err)
		assert.NotNil(t, enc)
	}
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	enc, err := NewAESEncrypter(newTestKey(t))
	require.NoError(t, err)

	ct, err := enc.Encrypt([]byte(""))
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ct)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}
