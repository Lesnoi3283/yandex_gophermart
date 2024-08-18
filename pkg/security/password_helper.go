package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

const passwordSaltLen int = 6

func HashPassword(password string, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	hasher.Write([]byte(salt))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CheckPassword returns TRUE if password is CORRECT
func CheckPassword(password string, passwordHash string, passwordSalt string) bool {
	return HashPassword(password, passwordSalt) == passwordHash
}

func GenPasswordSalt() (string, error) {
	bytes := make([]byte, passwordSaltLen)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", errors.Join(errors.New("error while generating password salt"), err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
