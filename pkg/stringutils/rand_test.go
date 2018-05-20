package stringutils

import "testing"

func TestSecureRandomAlphaString(t *testing.T) {
	randString := SecureRandomAlphaString(8)
	if len(randString) != 8 {
		t.Fail()
	}
}

func TestSecureRandomBytes(t *testing.T) {
	randBytes := secureRandomBytes(29)
	if len(randBytes) != 29 {
		t.Fail()
	}
}
