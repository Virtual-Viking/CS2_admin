package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"

)

const (
	gcmNonceSize = 12
	aesKeySize   = 32
)

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded string.
func Encrypt(plaintext string, key []byte) (string, error) {
	if len(key) != aesKeySize {
		return "", fmt.Errorf("key must be %d bytes", aesKeySize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded ciphertext using AES-256-GCM.
func Decrypt(ciphertext string, key []byte) (string, error) {
	if len(key) != aesKeySize {
		return "", fmt.Errorf("key must be %d bytes", aesKeySize)
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	if len(data) < gcmNonceSize+aesGCM.Overhead() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:gcmNonceSize], data[gcmNonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// DeriveKey derives a 32-byte key from machine-specific data.
// On Windows, uses hostname + MachineGuid from registry when available.
// Falls back to hostname + username hash on other platforms or when registry is unavailable.
func DeriveKey() ([]byte, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}

	seed := hostname
	if guid := getMachineGUID(); guid != "" {
		seed += guid
	} else {
		u, err := user.Current()
		if err == nil {
			h := sha256.Sum256([]byte(u.Username))
			seed += hex.EncodeToString(h[:])
		}
	}

	key := sha256.Sum256([]byte(seed))
	return key[:], nil
}

// GenerateRandomBytes returns n cryptographically random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("n must be positive")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("read random: %w", err)
	}
	return b, nil
}
