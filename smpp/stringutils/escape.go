package stringutils

import "strings"

func EscapeQuotes(args ...interface{}) []interface{} {
	for k, v := range args {
		switch v.(type) {
		case string:
			args[k] = strings.Replace(v.(string), "'", "\\'", -1)
		}
	}
	return args
}

func EscapeQuote(arg string) string {
	return strings.Replace(arg, "'", "\\'", -1)
}
