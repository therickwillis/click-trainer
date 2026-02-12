package rooms

import (
	"crypto/rand"
	"math/big"
)

// Alphabet excludes ambiguous characters: 0, O, 1, I, L
const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const codeLength = 4

func GenerateCode() (string, error) {
	code := make([]byte, codeLength)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		code[i] = alphabet[n.Int64()]
	}
	return string(code), nil
}
