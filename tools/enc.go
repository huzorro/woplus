package tools

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
)

func HmacSha1(message string, secret string) string {
	//hmac, use sha1
	key := []byte(secret)
	m := hmac.New(sha1.New, key)
	m.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}
