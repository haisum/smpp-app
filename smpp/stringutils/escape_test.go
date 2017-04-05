package stringutils

import (
	"testing"
	"reflect"
)

func TestEscapeQuote(t *testing.T) {
	testString := "I'm fine how're you"
	expectedString := `I\'m fine how\'re you`
	if EscapeQuote(testString) != expectedString {
		t.Errorf("Expected: %s. Got: %s", expectedString, EscapeQuote(testString))
	}
}

func TestEscapeQuotes(t *testing.T) {
	testStrings := []interface{}{"I'm fine how're you", 34, "hello's", 3.4, "pee;;''sdf"}
	expectedStrings := []interface{}{`I\'m fine how\'re you`, 34, `hello\'s`, 3.4, `pee;;\'\'sdf`}
	if !reflect.DeepEqual(expectedStrings, EscapeQuotes(testStrings...)) {
		t.Fail()
	}
}
