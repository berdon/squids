package idgen

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const base36Alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

func EncodeBase36(data []byte, length int) string {
	num := new(big.Int).SetBytes(data)
	var result strings.Builder
	base := big.NewInt(36)
	zero := big.NewInt(0)
	mod := new(big.Int)
	chars := make([]byte, 0, length)

	for num.Cmp(zero) > 0 {
		num.DivMod(num, base, mod)
		chars = append(chars, base36Alphabet[mod.Int64()])
	}
	for i := len(chars) - 1; i >= 0; i-- {
		result.WriteByte(chars[i])
	}
	str := result.String()
	if len(str) < length {
		str = strings.Repeat("0", length-len(str)) + str
	}
	if len(str) > length {
		str = str[len(str)-length:]
	}
	return str
}

func GenerateHashID(prefix, title, description, creator string, timestamp time.Time, length, nonce int) string {
	content := fmt.Sprintf("%s|%s|%s|%d|%d", title, description, creator, timestamp.UnixNano(), nonce)
	hash := sha256.Sum256([]byte(content))

	var numBytes int
	switch length {
	case 3:
		numBytes = 2
	case 4:
		numBytes = 3
	case 5:
		numBytes = 4
	case 6:
		numBytes = 4
	case 7:
		numBytes = 5
	case 8:
		numBytes = 5
	default:
		numBytes = 3
	}

	shortHash := EncodeBase36(hash[:numBytes], length)
	return fmt.Sprintf("%s-%s", prefix, shortHash)
}
