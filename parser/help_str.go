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

// 最后一个有意义的符号是否是start(后面跟踪回车、空格、注释不影响, 前面也不能有start)
func GetLastIsIdentifier(l []*word, start string) (bool, int) {
	ok := false
	i := 0
	var w *word
	for i, w = range l {
		if w.t == wordT_line {
			return ok, i
		} else if w.str == start {
			ok = true
		} else if !ok || !(w.t == wordT_doc || InArrString(w.str, []string{" ", "\t", "\n"})) {
			if ok {
				return false, i
			}
		}
	}

	return ok, i
}

// 获取第一个单词
func GetFistWord(l []*word) (string, int) {
	for i, w := range l {
		if w.t == wordT_word {
			return w.str, i
		}
	}
	return "", len(l)
}

// 获取某字符串后第一个单词
func GetFistWordBehindStr(l []*word, behind string) (string, int) {
	init := false
	for i, w := range l {
		if w.t == wordT_word {
			if init {
				return w.str, i
			} else if w.str == behind {
				init = true
			}
		}
	}
	return "", len(l)
}

// 括号引用起来的块, 或者第一行没有块就换行
func GetBracketsOrLn(l []*word, start, end string) (int, int, bool) {
	for i, w := range l {
		if w.t == wordT_line {
			return 0, i, false
		} else if w.t == wordT_division && w.str == start {
			startInt, endInt := GetBrackets(l, start, end)
			return startInt, endInt, true
		}
	}

	return 0, 0, false
}

// 括号引用起来的块, 词性必须是分隔符
func GetBrackets(l []*word, start, end string) (int, int) {
	var startInt, endInt int

	bCount := 0
	for i, w := range l {
		if bCount == 0 {
			if w.t == wordT_division && w.str == start {
				startInt = i
				bCount++
			}
		} else {
			if w.t == wordT_division {
				switch w.str {
				case end:
					bCount--
					if bCount <= 0 {
						endInt = i
						return startInt, endInt
					}
				case start:
					bCount++
				}
			}
		}
	}

	return startInt, endInt
}

// 组装成数组
func GetArrWord(l []*word) [][]*word {
	got := make([][]*word, 0)
	sl := make([]*word, 0)
	for _, w := range l {
		switch w.t {
		case wordT_word:
			sl = append(sl, w)
		case wordT_division:
			if !InArrString(w.str, []string{" ", "\t"}) {
				sl = append(sl, w)
			}
		case wordT_line:
			if len(sl) > 0 {
				got = append(got, sl)
			}
			sl = make([]*word, 0)
		}
	}
	if len(sl) > 0 {
		got = append(got, sl)
	}
	return got
}
