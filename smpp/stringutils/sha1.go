package stringutils

import (
	"crypto/sha1"
	"encoding/hex"
)

func ToSHA1(s string) string {
	sh := sha1.New()
	sh.Write([]byte(s))
	return hex.EncodeToString(sh.Sum(nil))
}
