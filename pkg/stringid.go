package stringid

import (
	"crypto/rand"
	"encoding/hex"
)

const shortLen = 12
func GenerateRandomID() string {
	b := make([]byte, 32)
	for {
		if _, err := rand.Read(b); err != nil {
			panic(err) // This shouldn't happen
		}
		return hex.EncodeToString(b)
	}
}
