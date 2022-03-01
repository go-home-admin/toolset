package parser

import (
	"unicode"
)

// 标识符
func IsIdentifier(r int32) bool {
	if unicode.IsLetter(r) {
		return true
	} else if unicode.IsDigit(r) {
		return true
	} else if 95 == r {
		return true
	}

	return false
}

// 判断是否以某字符串作为结尾 HasSuffix
func HasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// 判断是否以某字符串作为开始 HasPrefix
func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func InArrString(str string, arr []string) bool {
	for _, s := range arr {
		if s == str {
			return true
		}
	}
	return false
}
