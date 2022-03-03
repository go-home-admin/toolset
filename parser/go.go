package parser

import (
	"fmt"
	"strings"
)

/**
golang parser 非完整token实现
*/
type DirParser struct {
	PackageName string
	Imports     map[string]string
	Types       map[string]GoType
	Funds       map[string]string
}

func NewGoParser(path string) DirParser {
	d := DirParser{}
	for _, file := range loadGoFiles(path) {
		gof, _ := parserGoFile(file)
		fmt.Println(gof)
	}

	return d
}

func loadGoFiles(path string) []FileInfo {
	return loadFiles(path, ".go")
}

func parserGoFile(info FileInfo) (string, error) {
	d := DirParser{
		PackageName: "",
		Imports:     make(map[string]string),
		Types:       make(map[string]GoType),
		Funds:       make(map[string]string),
	}

	l := words(info.path)
	for offset := 0; offset < len(l.list); offset++ {
		fmt.Println(offset)
		work := l.list[offset]
		// 原则上, 每个块级别的作用域必须自己处理完, 返回的偏移必须是下一个块的开始
		switch work.t {
		case wordT_line:
		case wordT_division:
		case wordT_doc:
		case wordT_word:
			switch work.str {
			case "package":
				d.PackageName, offset = handlePackageName(l.list, offset)
			case "import":
				var imap map[string]string
				imap, offset = handleImports(l.list, offset)
				for k, v := range imap {
					d.Imports[k] = v
				}
			case "type":
				var imap GoType
				imap, offset = handleTypes(l.list, offset)
				d.Types[imap.Name] = imap
			case "func":
				var imap map[string]string
				imap, offset = handleFunds(l.list, offset)
				for k, v := range imap {
					d.Imports[k] = v
				}
			case "const":
				_, offset = handleCosts(l.list, offset)
			case "var":
				_, offset = handleVars(l.list, offset)
			default:

			}
		}
	}

	return "", nil
}

func handlePackageName(l []*word, offset int) (string, int) {
	newOffset := offset
	name := ""
	for i, w := range l[offset:] {
		if w.t == wordT_line {
			name = l[i-1].str
			newOffset = i
			break
		}
	}

	return name, newOffset
}

func handleImports(l []*word, offset int) (map[string]string, int) {
	newOffset := offset
	imap := make(map[string]string)
	var key, val string
	start := 0

gofer:
	for i, w := range l[offset+1:] {
		switch w.t {
		case wordT_line:
			newOffset = i + offset + 1
			switch start {
			case 0:
				break
			case 1:
				if l[newOffset+1].str == ")" {
					i = newOffset + 1
					break gofer
				}
				key, val = "", ""
			}
		case wordT_word:
			if w.str[0:1] == "\"" {
				val = w.str[1 : len(w.str)-1]

				if key == "" {
					temp := strings.Split(val, "/")
					key = temp[len(temp)-1]
				}
				imap[key] = val
			} else {
				key = w.str
			}
		case wordT_division:
			if w.str == "(" {
				start = 1
			}
		}
	}

	return imap, newOffset + 1
}

type GoType struct {
	Doc   string
	Name  string
	Attrs map[string]GoTypeAttr
}
type GoTypeAttr struct {
	Name       string
	TypeName   string
	TypeAlias  string
	TypeImport string
	Tag        map[string]string
}

// 普通指针
func (receiver GoTypeAttr) IsPointer() bool {
	return receiver.TypeName[0:1] == "*"
}

// 组装成数组, 只限name type other\n结构
func getArrGoWord(l []*word) [][]string {
	got := make([][]string, 0)
	arr := GetArrWord(l)
	for _, i := range arr {
		lis := i[len(i)-1].str
		if lis[0:1] == "`" && len(i) >= 3 {
			ty := ""
			for in := 1; in < len(i)-1; in++ {
				if i[in].t != wordT_doc {
					ty = ty + i[in].str
				}
			}
			got = append(got, []string{i[0].str, ty, lis})
		}
	}

	return got
}
func handleTypes(l []*word, offset int) (GoType, int) {
	newOffset := offset
	nl := l[offset:]
	got := GoType{
		Doc:   "",
		Name:  "",
		Attrs: map[string]GoTypeAttr{},
	}
	ok, off := GetLastIsIdentifier(nl, "{")

	if ok {
		// 新结构
		var i int
		got.Name, i = GetFistWordBehindStr(nl, "type")
		nl = nl[i+1:]
		st, et := GetBrackets(nl, "{", "}")
		newOffset = offset + off + i + st + 1
		nl := nl[st+1 : et]
		arrLn := getArrGoWord(nl)
		for _, wordAttrs := range arrLn {
			// 获取属性信息
			// TODO 当前仅支持有注解的
			if len(wordAttrs) == 3 && strings.Index(wordAttrs[2], "`") != 0 {
				attr := GoTypeAttr{
					Name:       wordAttrs[0],
					TypeName:   wordAttrs[1],
					TypeAlias:  "",
					TypeImport: "",
					Tag:        map[string]string{},
				}
				// 解析 go tag

				got.Attrs[attr.Name] = attr
			}
		}
	} else {
		// struct 别名
		// TODO 目前不需要支持
		newOffset = off + offset
	}

	return got, newOffset
}
func handleFunds(l []*word, offset int) (map[string]string, int) {
	ok, off := GetLastIsIdentifier(l[offset:], "(")
	if ok {
		return nil, off + offset
	}
	_, et := GetBrackets(l[offset:], "(", ")")
	return nil, offset + et
}
func handleCosts(l []*word, offset int) (map[string]string, int) {
	ok, off := GetLastIsIdentifier(l[offset:], "(")
	if ok {
		return nil, off + offset
	}
	_, et := GetBrackets(l[offset:], "(", ")")
	return nil, offset + et
}

func handleVars(l []*word, offset int) (map[string]string, int) {
	ok, off := GetLastIsIdentifier(l[offset:], "(")
	if ok {
		return nil, off + offset
	}
	_, et := GetBrackets(l[offset:], "(", ")")
	return nil, offset + et
}
