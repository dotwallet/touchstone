package util

import (
	"encoding/json"
	"math/rand"
	"time"
)

const (
	LETTER_BYTES = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func SplitString(s string, l int) []string {
	result := make([]string, 0, 4)
	resultLen := 0
	for resultLen+l < len(s) {
		subs := s[resultLen : resultLen+l]
		result = append(result, subs)
		resultLen += l
	}
	if resultLen != len(s) {
		subs := s[resultLen:]
		result = append(result, subs)
	}
	return result
}

func ToJson(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func RandStringBytes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = LETTER_BYTES[rand.Intn(len(LETTER_BYTES))]
	}
	return string(b)
}
