package util

import (
	"crypto/md5"
	"encoding/hex"
)

func GenerateId(in string) string {
	binHash := md5.Sum([]byte(in))
	return hex.EncodeToString(binHash[:])
}
