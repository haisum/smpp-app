package stringutils

import "testing"

func TestToSHA1(t *testing.T) {
	word := "qwerty"
	hash := "b1b3773a05c0ed0176787a4f1574ff0075f7521e"
	if ToSHA1(word) != hash {
		t.Errorf("exepected: %s, got: %s", hash, ToSHA1(word))
	}
}
