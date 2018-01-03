package smtext

import "github.com/fiorix/go-smpp/smpp/pdu/pdutext"

const (
	// EncUCS is string representation of ucs encoding
	EncUCS = "ucs"
	// EncLatin is string representation of latin encoding
	EncLatin = "latin"
)

// Total counts number of messages in one text string
func Total(msg, enc string) int {
	var text pdutext.Codec
	if enc == EncUCS {
		text = pdutext.UCS2(msg)
	} else {
		text = pdutext.Raw(msg)
	}
	maxLen := 134 // 140-6 (UDH)
	rawMsg := text.Encode()
	return int(len(rawMsg)/maxLen) + 1
}

// IsASCII checks if given string is ascii characters only
func IsASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return false
		}
	}
	return true
}
