package parser

import (
	"fmt"
	"os"
	"strings"
)

// GoFileParser 非完整token实现
type GoFileParser struct {
	PackageName string
	PackageDoc  string
	Imports     map[string]string
	Types       map[string]GoType
	Funds       map[string]GoFunc
}

func NewGoParserForDir(path string) map[string][]GoFileParser {
	got := make(map[string][]GoFileParser)
	for _, dir := range GetChildrenDir(path) {
		arr := make([]GoFileParser, 0)
		for _, file := range dir.GetFiles(".go") {
			gof, _ := GetFileParser(file.Path)
			arr = append(arr, gof)
		}
		got[dir.Path] = arr
	}

	return got
}

// go 关键字语法块
// map\[]\interface{}
func getWordsWitchGo(l *GoWords) GoWords {
	var got = GoWords{
		list: make([]*word, 0),
	}

	for offset := 0; offset < len(l.list); offset++ {
		work := l.list[offset]
		switch work.Ty {
		case wordT_word:
			switch work.Str {
			case "interface":
				if len(l.list) >= (offset+2) && l.list[offset+1].Str == "{" && l.list[offset+2].Str == "}" {
					offset = offset + 2
					work.Str = work.Str + "{}"
					got.list = append(got.list, work)
				} else {
					got.list = append(got.list, work)
				}
			default:
				got.list = append(got.list, work)
			}
		default:
			got.list = append(got.list, work)
		}
	}
	return got
}

func GetFileParser(path string) (GoFileParser, error) {
	d := &GoFileParser{
		PackageName: "",
		PackageDoc:  "",
		Imports:     make(map[string]string),
		Types:       make(map[string]GoType),
		Funds:       make(map[string]GoFunc),
	}

	l := getWordsWitchFile(path)
	l = getWordsWitchGo(&l)
	lastDoc := ""
	for offset := 0; offset < len(l.list); offset++ {
		work := l.list[offset]
		// 原则上, 每个块级别的作用域必须自己处理完, 返回的偏移必须是下一个块的开始
		switch work.Ty {
		case wordT_line:
			lastDoc += work.Str
		case wordT_division:
		case wordT_doc:
			lastDoc += work.Str
		case wordT_word:
			switch work.Str {
			case "package":
				d.PackageDoc = lastDoc
				d.PackageName, offset = handlePackageName(l.list, offset)
				lastDoc = ""
			case "import":
				var imap map[string]string
				imap, offset = handleImports(l.list, offset)
				for k, v := range imap {
					d.Imports[k] = v
				}
				lastDoc = ""
			case "type":
				var imap GoType
				imap, offset = handleTypes(l.list, offset, d)
				imap.Doc = GoDoc(lastDoc)
				d.Types[imap.Name] = imap
				lastDoc = ""
			case "func":
				var gf GoFunc
				gf, offset = handleFunds(l.list, offset)
				d.Funds[gf.Name] = gf
				lastDoc = ""
			case "const":
				_, offset = handleCosts(l.list, offset)
				lastDoc = ""
			case "var":
				_, offset = handleVars(l.list, offset)
				lastDoc = ""
			default:
				// 遇到未支持的结构, 直接跳到\n}\n重新开始
				endCheck := 0
				for offset < len(l.list) {
					offset++
					work2 := l.list[offset]
					if endCheck == 0 && work2.Ty == wordT_line {
						endCheck++
					} else if endCheck == 1 && (work2.Str == ")" || work2.Str == "}") {
						endCheck++
					} else if endCheck == 2 && work2.Ty == wordT_line {
						break
					} else {
						endCheck = 0
					}
				}

				nl := l.list[offset:]
				str := ""
				for _, w := range nl {
					str += w.Str
				}
				fmt.Println("文件块作用域似乎解析有错误\n", path, "\n", offset, "\n", str)
				os.Exit(1)
			}
		}
	}

	return *d, nil
}

func handlePackageName(l []*word, offset int) (string, int) {
	name, i := GetFistWordBehindStr(l[offset:], "package")
	return name, offset + i
}

func getImport(sl []string) (string, string) {
	if len(sl) == 2 {
		return sl[0], sl[1][1 : len(sl[1])-1]
	}

	str := sl[0][1 : len(sl[0])-1]
	temp := strings.Split(str, "/")
	key := temp[len(temp)-1]
	return key, str
}

func handleImports(l []*word, offset int) (map[string]string, int) {
	newOffset := offset
	imap := make(map[string]string)
	var key, val string

	ft, fti := GetFistStr(l[offset+1:])
	if ft != "(" {
		arr := make([]string, 0)
		for i, w := range l[offset+fti:] {
			if wordT_line == w.Ty {
				newOffset = offset + fti + i
				key, val = getImport(arr)
				imap[key] = val
				return imap, newOffset
			}

			if w.Ty == wordT_word {
				arr = append(arr, w.Str)
			}
		}
	} else {
		st, et := GetBrackets(l[offset+1:], "(", ")")
		st = st + offset + 1
		et = et + offset + 1
		newOffset = et

		arr := make([]string, 0)
		for _, w := range l[st : et+1] {
			if wordT_line == w.Ty && len(arr) != 0 {
				key, val = getImport(arr)
				imap[key] = val
				arr = make([]string, 0)
			}

			if w.Ty == wordT_word {
				arr = append(arr, w.Str)
			}
		}
	}
	return imap, newOffset
}

type GoType struct {
	Doc          GoDoc
	Name         string
	Attrs        map[string]GoTypeAttr
	AttrsSort    []string
	GoFileParser *GoFileParser
}

type GoTypeAttr struct {
	Name         string
	TypeName     string
	TypeAlias    string
	TypeImport   string
	InPackage    bool // 是否本包的引用
	Tag          map[string]TagDoc
	GoFileParser *GoFileParser
}

type TagDoc string

func (t TagDoc) Get(num int) string {
	s := string(t)
	sr := strings.Split(s, ",")
	return strings.Trim(sr[num], " ")
}

func (t TagDoc) Count() int {
	s := string(t)
	sr := strings.Split(s, ",")
	return len(sr)
}

type GoDoc string

// HasAnnotation 是否存在某个注解
func (d GoDoc) HasAnnotation(check string) bool {
	return strings.Index(string(d), check) != -1
}

func (d GoDoc) GetAlias() string {
	ns := strings.ReplaceAll(string(d), "//", " ")
	l := GetWords(ns)
	num := 0
	for i, w := range l {
		if w.Ty == wordT_word {
			if w.Str == "Bean" {
				if l[i-1].Str == "@" {
					num = i
				}
			} else if num == -1 {
				return w.Str[1 : len(w.Str)-1]
			}
		} else if num == (i-1) && w.Str == "(" {
			num = -1
		}
	}
	return ""
}

// IsPointer 普通指针
func (receiver GoTypeAttr) IsPointer() bool {
	return receiver.TypeName[0:1] == "*"
}

func (receiver GoTypeAttr) HasTag(name string) bool {
	for s := range receiver.Tag {
		if s == name {
			return true
		}
	}
	return false
}

// 组装成数组, 只限name type other\n结构
func getArrGoWord(l []*word) [][]string {
	got := make([][]string, 0)
	arr := GetArrWord(l)
	for _, i := range arr {
		lis := i[len(i)-1].Str
		if lis[0:1] == "`" && len(i) >= 2 {
			ty := ""
			for in := 1; in < len(i)-1; in++ {
				if i[in].Ty != wordT_doc {
					ty = ty + i[in].Str
				}
			}
			if ty == "" || i[0].Str == "*" {
				ty = i[0].Str + ty
				arr2 := strings.Split(ty, ".")
				name := arr2[len(arr2)-1]
				got = append(got, []string{strings.Trim(name, "*"), ty, lis})
			} else {
				got = append(got, []string{i[0].Str, ty, lis})
			}
		}
	}

	return got
}

// 把go结构的tag格式化成数组 source = `inject:"" json:"orm"`
func getArrGoTag(source string) [][]string {
	tagStr := source[1 : len(source)-1]
	wl := GetWords(tagStr)
	// 每三个一组
	i := 0
	got := make([][]string, 0)
	arr := make([]string, 0)
	for _, w := range wl {
		if w.Ty == wordT_word {
			arr = append(arr, w.Str)
			i++
			if i >= 2 {
				i = 0
				got = append(got, arr)
				arr = make([]string, 0)
			}
		}
	}

	return got
}
func handleTypes(l []*word, offset int, d *GoFileParser) (GoType, int) {
	newOffset := offset
	nl := l[offset:]
	got := GoType{
		Doc:          "",
		Name:         "",
		Attrs:        map[string]GoTypeAttr{},
		AttrsSort:    make([]string, 0),
		GoFileParser: d,
	}
	ok, off := GetLastIsIdentifier(nl, "{")
	if ok {
		// 新结构
		var i int
		got.Name, i = GetFistWordBehindStr(nl, "type")
		nl = nl[i+1:]
		st, et := GetBrackets(nl, "{", "}")
		newOffset = offset + i + et + 1
		nl := nl[st+1 : et]
		arrLn := getArrGoWord(nl)
		for _, wordAttrs := range arrLn {
			if strings.Index(wordAttrs[len(wordAttrs)-1], "`") != 0 {
				break
			}
			// 获取属性信息
			attr := GoTypeAttr{GoFileParser: d}
			for i := len(wordAttrs) - 1; i >= 0; i-- {
				s := wordAttrs[i]
				if attr.Tag == nil {
					// 解析 go tag
					attr.Tag = make(map[string]TagDoc)
					for _, tagStrArr := range getArrGoTag(s) {
						td := tagStrArr[1]
						attr.Tag[tagStrArr[0]] = TagDoc(td[1 : len(td)-1])
					}
				} else if attr.TypeName == "" {
					attr.TypeName = s
					getTypeAlias(s, d, &attr)
				} else {
					attr.Name = s
				}
			}
			got.Attrs[attr.Name] = attr
			got.AttrsSort = append(got.AttrsSort, attr.Name)
		}
	} else {
		// struct 别名
		got.Name, _ = GetFistWordBehindStr(nl, "type")
		newOffset = off + offset
	}

	return got, newOffset
}

// 根据属性声明类型或者类型的引入名称
func getTypeAlias(str string, d *GoFileParser, attr *GoTypeAttr) {
	wArr := GetWords(str)
	wf := wArr[0]

	if wf.Ty == wordT_word || wf.Str == "*" {
		if (wf.Str == "*" && len(wArr) >= 3) || (wf.Str != "*" && len(wArr) >= 2) {
			attr.TypeAlias, _ = GetFistWord(wArr)
			attr.TypeImport = d.Imports[attr.TypeAlias]
			return
		}
	}
	// 本包
	attr.TypeAlias = d.PackageName
	attr.TypeImport = "" // TODO
	attr.InPackage = true
}

type GoFunc struct {
	Name string
	Stu  string
}

func handleFunds(l []*word, offset int) (GoFunc, int) {
	ft, _ := GetFistStr(l[offset+1:])
	name := ""
	if ft != "(" {
		// 普通函数
		var i int
		name, i = GetFistWordBehindStr(l[offset:], "func")
		offset = offset + i
		_, et := GetBrackets(l[offset:], "(", ")")
		offset = offset + et
	} else {
		// 结构函数
		_, et := GetBrackets(l[offset:], "(", ")")
		offset = offset + et
		name, _ = GetFistWord(l[offset:])
		_, et = GetBrackets(l[offset:], "(", ")")
		offset = offset + et
	}
	// 排除返回值的interface{}
	st, et := GetBrackets(l[offset:], "{", "}")
	interCount := 0
	for _, w := range l[offset : offset+st] {
		if w.Str == "interface" {
			interCount++
		}
	}

	if interCount != 0 {
		for i := 0; i <= interCount; i++ {
			_, et := GetBrackets(l[offset:], "{", "}")
			offset = offset + et
		}
	} else {
		offset = offset + et
	}

	return GoFunc{Name: name}, offset + 1
}
func handleCosts(l []*word, offset int) (map[string]string, int) {
	return handleVars(l, offset)
}

func handleVars(l []*word, offset int) (map[string]string, int) {
	endCheck := 0
	offset++
	for offset < len(l)-1 {
		offset++
		work2 := l[offset]
		if endCheck == 0 && work2.Ty == wordT_line {
			endCheck++
		} else if endCheck == 1 && !(work2.Str == " " || work2.Str == "\t") {
			endCheck++
		} else if endCheck == 2 && work2.Ty == wordT_line {
			break
		} else {
			endCheck = 0
		}
	}
	return nil, offset
}

func toStr(l []*word) string {
	s := ""
	for _, w := range l {
		s += w.Str
	}
	return s
}
