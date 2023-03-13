package parser

import (
	"sort"
	"strconv"
	"strings"
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
		if w.Ty == wordT_line {
			return ok, i
		} else if w.Str == start {
			ok = true
		} else if ok && !(w.Ty == wordT_doc || InArrString(w.Str, []string{" ", "\t"})) {
			ok = false
		}
	}

	return ok, i
}

// 获取第一个单词
func GetFistWord(l []*word) (string, int) {
	for i, w := range l {
		if w.Ty == wordT_word {
			return w.Str, i
		}
	}
	return "", len(l)
}

// 获取第n个单词
func GetWord(l []*word, n int) (string, int) {
	for i, w := range l {
		if w.Ty == wordT_word {
			n--
			if n <= 0 {
				return w.Str, i
			}
		}
	}
	return "", len(l)
}

// 获取第一个字符(不包括空格, 换行符, 制表)
func GetFistStr(l []*word) (string, int) {
	for i, w := range l {
		if !InArrString(w.Str, []string{" ", "\n", "\t"}) {
			return w.Str, i
		}
	}
	return "", len(l)
}

// 获取换行前所有词
func GetStrAtEnd(l []*word) (string, int) {
	s := ""
	for i, w := range l {
		if w.Str == "\n" {
			return s, i
		} else if w.Str != " " {
			s = s + w.Str
		}
	}
	return "", len(l)
}

// 获取下一行开始
func NextLine(l []*word) int {
	for i, w := range l {
		if w.Ty == wordT_line {
			return i
		}
	}
	return len(l)
}

// 获取某字符串后第一个单词
func GetFistWordBehindStr(l []*word, behind string) (string, int) {
	init := false
	for i, w := range l {
		if w.Ty == wordT_word {
			if init {
				return w.Str, i
			} else if w.Str == behind {
				init = true
			}
		}
	}
	return "", len(l)
}

// 括号引用起来的块, 或者第一行没有块就换行
func GetBracketsOrLn(l []*word, start, end string) (int, int, bool) {
	for i, w := range l {
		if w.Ty == wordT_line {
			return 0, i, false
		} else if w.Ty == wordT_division && w.Str == start {
			startInt, endInt := GetBrackets(l, start, end)
			return startInt, endInt, true
		}
	}

	return 0, 0, false
}

// 括号引用起来的块, 不限制分隔符
func GetBracketsAtString(l []*word, start, end string) (int, int) {
	var startInt, endInt int

	bCount := 0
	for i, w := range l {
		if bCount == 0 {
			if w.Str == start {
				startInt = i
				bCount++
			}
		} else {
			switch w.Str {
			case start:
				bCount++
			case end:
				bCount--
				if bCount <= 0 {
					endInt = i
					return startInt, endInt
				}
			}
		}
	}

	return startInt, endInt
}

// 括号引用起来的块, 词性必须是分隔符
// 返回是开始偏移和结束偏移
func GetBrackets(l []*word, start, end string) (int, int) {
	var startInt, endInt int

	bCount := 0
	for i, w := range l {
		if bCount == 0 {
			if w.Ty == wordT_division && w.Str == start {
				startInt = i
				bCount++
			}
		} else {
			if w.Ty == wordT_division {
				switch w.Str {
				case start:
					bCount++
				case end:
					bCount--
					if bCount <= 0 {
						endInt = i
						return startInt, endInt
					}
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
		switch w.Ty {
		case wordT_word:
			sl = append(sl, w)
		case wordT_division:
			if !InArrString(w.Str, []string{" ", "\t"}) {
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

// 对原始字符串分词
func GetWords(source string) []*word {
	status := scannerStatus_NewLine
	work := ""
	lastIsSpe := false
	list := make([]*word, 0)
	for _, s := range source {
		str := string(s)
		stop := false
		switch status {
		case scannerStatus_Doc:
			work = work + str
			stop = true
		case scannerStatus_Doc2:
			work = work + str
			stop = true
			// 检查是否*/结束了
			if str == "/" && HasSuffix(work, "*/") {
				list = append(list, &word{
					Str: work,
					Ty:  wordT_doc,
				})
				// 分割后从新开始
				work = ""
				status = scannerStatus_NewWork
			}
		case scannerStatus_DocWait:
			switch str {
			case "/":
				work = work + str
				status = scannerStatus_Doc
				stop = true
			case "*":
				work = work + str
				status = scannerStatus_Doc2
				stop = true
			default:
				// 没有进入文档模式, 那么上一个就是分割符号
				list = append(list, &word{
					Str: work,
					Ty:  wordT_division,
				})
				// 分割后从新开始
				work = ""
				status = scannerStatus_NewWork
			}
		case scannerStatus_quote:
			work = work + str
			stop = true
			if str == "\"" && !HasSuffix(work, "\\\"") {
				list = append(list, &word{
					Str: work,
					Ty:  wordT_word,
				})
				// 分割后从新开始
				work = ""
				status = scannerStatus_NewWork
			}
		case scannerStatus_quote2:
			work = work + str
			stop = true
			if str == "'" {
				list = append(list, &word{
					Str: work,
					Ty:  wordT_word,
				})
				// 分割后从新开始
				work = ""
				status = scannerStatus_NewWork
			}
		case scannerStatus_quote3:
			work = work + str
			stop = true
			if str == "`" {
				list = append(list, &word{
					Str: work,
					Ty:  wordT_word,
				})
				// 分割后从新开始
				work = ""
				status = scannerStatus_NewWork
			}
		case scannerStatus_NewLine, scannerStatus_NewWork:
			switch str {
			case "/":
				work = work + str
				status = scannerStatus_DocWait
				stop = true
			case "\"":
				work = work + str
				status = scannerStatus_quote
				stop = true
			case "'":
				work = work + str
				status = scannerStatus_quote2
				stop = true
			case "`":
				work = work + str
				status = scannerStatus_quote3
				stop = true
			}
		}
		if !stop {
			if IsIdentifier(s) {
				// 标识符: 字母, 数字, _
				work = work + str
				status = scannerStatus_Work
				lastIsSpe = false
			} else if InArrString(str, []string{" ", "\t"}) {
				// 合并多余的空格
				if !lastIsSpe {
					if len(work) != 0 {
						list = append(list, &word{
							Str: work,
							Ty:  wordT_word,
						})
					}
					list = append(list, &word{
						Str: str,
						Ty:  wordT_division,
					})
					work = ""
					status = scannerStatus_NewWork
					lastIsSpe = true
				}
			} else {
				if len(work) != 0 {
					list = append(list, &word{
						Str: work,
						Ty:  wordT_word,
					})
				}
				list = append(list, &word{
					Str: str,
					Ty:  wordT_division,
				})
				work = ""
				status = scannerStatus_NewWork
				lastIsSpe = false
			}
		} else {
			lastIsSpe = false
		}
	}

	if status == scannerStatus_Work && len(work) != 0 {
		list = append(list, &word{
			Str: work,
			Ty:  wordT_word,
		})
	}

	return list
}

// 驼峰转蛇形
func StringToSnake(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:]))
}

// 蛇形转驼峰
func StringToHump(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && (d == '_' || d == '-') && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:])
}

// SortMap 排序map
func SortMap(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetImportStrForMap 生成import
func GetImportStrForMap(m map[string]string) string {
	sk := SortMap(m)
	got := ""
	for _, k := range sk {
		got += "\n\t" + m[k] + " \"" + k + "\""
	}

	return got
}

// GenImportAlias 生成 import => alias
func GenImportAlias(path, packageName string, m map[string]string) map[string]string {
	aliasMapImport := make(map[string]string)
	importMapAlias := make(map[string]string)

	keys := make([]string, 0)
	for s, _ := range m {
		keys = append(keys, s)
	}
	sort.Strings(keys)
	for _, k := range keys {
		imp := m[k]
		temp := strings.Split(imp, "/")
		key := temp[len(temp)-1]

		if _, ok := aliasMapImport[key]; ok {
			for i := 1; i < 1000; i++ {
				newKey := key + strconv.Itoa(i)
				if _, ok2 := aliasMapImport[newKey]; !ok2 {
					key = newKey
					break
				}
			}
		}
		if key == packageName {
			if "/bootstrap/providers" != path {
				aliasMapImport[key+"_2"] = imp
			}
		} else {
			aliasMapImport[key] = imp
		}
	}
	for s, s2 := range aliasMapImport {
		importMapAlias[s2] = s
	}

	return importMapAlias
}
