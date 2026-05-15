// Package aibot provides a Go SDK for WeChat Work (WeCom) AI Bot WebSocket API.
package aibot

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// GenerateReqID generates a unique request ID with the given prefix,
// formatted as "{prefix}_{timestamp}_{random}".
func GenerateReqID(prefix string) string {
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixMilli(), generateRandomString(8))
}

func generateReqID(prefix string) string {
	return GenerateReqID(prefix)
}

func stringsHasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}
