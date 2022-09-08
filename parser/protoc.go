package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ProtocFileParser 解释proto文件结构
type ProtocFileParser struct {
	Doc         string
	Syntax      string
	PackageName string
	Imports     []string
	Option      map[string]Option
	Services    map[string]Service
	Messages    map[string]Message
	Enums       map[string]Enum
}

type Option struct {
	Doc   string
	Key   string
	Val   string
	wl    []*word
	alias string
}

type Service struct {
	Protoc *ProtocFileParser
	Doc    string
	Name   string
	Opt    map[string]Option
	Rpc    map[string]ServiceRpc
}

type ServiceRpc struct {
	Doc    string
	Name   string
	Param  string
	Return string
	Opt    map[string]Option
}

type Message struct {
	Doc  string
	Name string
	Attr []Attr
	Opt  map[string]Option
}
type Attr struct {
	Doc      string
	Name     string
	Ty       string
	Num      int
	Repeated bool
	Message  *Message
}

type Enum struct {
	Doc  string
	Name string
	Opt  []Attr
}

func NewProtocParserForDir(path string) map[string][]ProtocFileParser {
	got := make(map[string][]ProtocFileParser)
	for _, dir := range GetChildrenDir(path) {
		arr := make([]ProtocFileParser, 0)
		for _, file := range dir.GetFiles(".proto") {
			gof, _ := GetProtoFileParser(file.Path)
			arr = append(arr, gof)
		}
		got[dir.Path] = arr
	}

	return got
}

func GetProtoFileParser(path string) (ProtocFileParser, error) {
	d := ProtocFileParser{
		Imports:  make([]string, 0),
		Option:   make(map[string]Option),
		Services: make(map[string]Service),
		Messages: make(map[string]Message),
		Enums:    make(map[string]Enum),
	}

	l := getWordsWitchFile(path)
	lastDoc := ""
	for offset := 0; offset < len(l.list); offset++ {
		work := l.list[offset]
		// 原则上, 每个块级别的作用域必须自己处理完, 返回的偏移必须是下一个块的开始
		switch work.Ty {
		case wordT_line:
		case wordT_division:
		case wordT_doc:
			lastDoc = work.Str
		case wordT_word:
			switch work.Str {
			case "syntax":
				d.Doc = doc(lastDoc)
				d.Syntax, offset = protoSyntax(l.list, offset)
				lastDoc = ""
			case "package":
				d.PackageName, offset = protoPackageName(l.list, offset)
				lastDoc = ""
			case "import":
				var imports string
				imports, offset = protoImport(l.list, offset)
				d.Imports = append(d.Imports, imports)
				lastDoc = ""
			case "option":
				var val Option
				val.Doc = doc(lastDoc)
				val, offset = protoOption(l.list, offset)
				d.Option[val.Key] = val
				lastDoc = ""
			case "service":
				var val Service
				val, offset = protoService(l.list, offset)
				val.Protoc = &d
				val.Doc = doc(lastDoc)
				d.Services[val.Name] = val
				lastDoc = ""
			case "message":
				var val Message
				val, offset = protoMessage(l.list, offset)
				val.Doc = doc(lastDoc)
				d.Messages[val.Name] = val
				lastDoc = ""
			case "enum":
				var val Enum
				val, offset = protoEnum(l.list, offset)
				val.Doc = doc(lastDoc)
				d.Enums[val.Name] = val
				lastDoc = ""
			case "extend":
				_, offset = protoExtend(l.list, offset)
				lastDoc = ""
			default:
				fmt.Println("文件块作用域似乎解析有错误", path, work.Str, offset)
			}
		}
	}

	return d, nil
}
func doc(doc string) string {
	doc = strings.TrimFunc(doc, IsSpaceAndEspecially)
	return doc
}

func IsSpaceAndEspecially(r rune) bool {
	// This property isn't the same as Z; special-case it.
	if uint32(r) <= unicode.MaxLatin1 {
		switch r {
		case '=', ';', '/', '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
			return true
		}
		return false
	}
	return false
}

func protoSyntax(l []*word, offset int) (string, int) {
	name, i := GetFistWordBehindStr(l[offset:], "syntax")
	return name[1 : len(name)-1], offset + i
}
func protoPackageName(l []*word, offset int) (string, int) {
	s, e := GetBracketsAtString(l[offset:], "package", ";")
	str := ""
	for _, w := range l[offset+s+1 : offset+e] {
		if w.Str != " " {
			str = str + w.Str
		}
	}
	return str, offset + e + 1
}

func protoImport(l []*word, offset int) (string, int) {
	name, i := GetFistWordBehindStr(l[offset:], "import")
	return name[1 : len(name)-1], offset + i
}
func protoOption(l []*word, offset int) (Option, int) {
	key, i := GetFistWordBehindStr(l[offset:], "option")
	offset = offset + i + 1
	val, i := GetFistWord(l[offset:])
	return Option{Key: key, Val: val[1 : len(val)-1]}, offset + i
}
func serverOption(l []*word, offset int) (Option, int) {
	st, et := GetBrackets(l[offset:], "(", ")")
	wl := l[offset+st : offset+et+1]
	offset = offset + et + 1
	val, i := GetFistWord(l[offset:])
	var key string
	var alias string
	if len(wl) >= 5 {
		alias = wl[1].Str
	}
	for _, w := range wl {
		key = key + w.Str
	}

	return Option{
		Key:   key[1 : len(key)-1],
		Val:   val[1 : len(val)-1],
		wl:    wl,
		alias: alias,
	}, offset + i
}
func protoService(l []*word, offset int) (Service, int) {
	name, i := GetFistWordBehindStr(l[offset:], "service")
	offset = offset + i
	st, et := GetBrackets(l[offset:], "{", "}")
	newOffset := offset + et
	nl := l[offset+st : offset+et]

	got := Service{
		Name: name,
		Opt:  make(map[string]Option, 0),
		Rpc:  make(map[string]ServiceRpc, 0),
	}
	doc := ""
	for offset := 0; offset < len(nl); offset++ {
		work := nl[offset]
		switch work.Ty {
		case wordT_line:
		case wordT_division:
		case wordT_doc:
			doc += work.Str
		case wordT_word:
			switch work.Str {
			case "option":
				var val Option
				val, offset = serverOption(nl, offset)
				got.Opt[val.Key] = val
			case "rpc":
				var val ServiceRpc
				val, offset = protoRpc(nl, offset)
				val.Doc = strings.ReplaceAll(doc, "//", "")
				got.Rpc[val.Name] = val
				doc = ""
			}
		}
	}

	return got, newOffset
}

func protoRpc(l []*word, offset int) (ServiceRpc, int) {
	name, i := GetFistWordBehindStr(l[offset:], "rpc")
	offset = offset + i + 1
	start, end, ok := GetBracketsOrLn(l[offset:], "(", ")")
	Param := ""
	if ok {
		ParamLW := l[offset+start+1 : offset+end]
		for _, w := range ParamLW {
			Param = Param + w.Str
		}
	}
	offset = offset + end + 1
	start, end, ok = GetBracketsOrLn(l[offset:], "(", ")")
	Return := ""
	if ok {
		ReturnLW := l[offset+start+1 : offset+end]
		for _, w := range ReturnLW {
			Return = Return + w.Str
		}
	}

	offset = offset + end + 1
	// opt
	opt := make(map[string]Option)
	st, et := GetBrackets(l[offset:], "{", "}")
	newOffset := offset + et + 1
	nl := l[offset+st : newOffset]
	for offset := 0; offset < len(nl); offset++ {
		work := nl[offset]
		switch work.Ty {
		case wordT_line:
		case wordT_division:
		case wordT_doc:
		case wordT_word:
			switch work.Str {
			case "option":
				var val Option
				val, offset = serverOption(nl, offset)
				opt[val.Key] = val
			}
		}
	}

	return ServiceRpc{
		Name:   name,
		Param:  Param,
		Return: Return,
		Opt:    opt,
	}, newOffset
}

func protoMessage(l []*word, offset int) (Message, int) {
	name, i := GetFistWordBehindStr(l[offset:], "message")
	offset = offset + i
	st, et := GetBrackets(l[offset:], "{", "}")
	newOffset := offset + et
	nl := l[offset+st+1 : offset+et]

	got := Message{
		Name: name,
		Attr: make([]Attr, 0),
		Opt:  nil,
	}

	attr := Attr{}
	for offset := 0; offset < len(nl); offset++ {
		work := nl[offset]
		switch work.Ty {
		case wordT_word:
			if attr.Ty == "" {
				switch work.Str {
				case "message":
					attr.Ty = "message"
					fallthrough
				case "enum":
					attr.Ty = "enum"
					fallthrough
				case "oneof":
					if attr.Ty == "" {
						attr.Ty = "oneof"
					}
					attr.Name, _ = GetFistWord(nl[offset+1:])
					st, et := GetBrackets(nl[offset:], "{", "}")
					attr.Message = protoOtherMessage(attr.Name, nl[offset+st:offset+et+1])
					attr.Doc = doc(attr.Doc)
					got.Attr = append(got.Attr, attr)
					attr = Attr{}
					offset = offset + et + 1
				case "repeated": // 重复的
					attr.Repeated = true
				case "reserved": // 保留标识符
				default: // case "double", "float", "int32", "int64", "unit32", "unit64", "fixed32", "fixed64", "sfixed32", "sfixed64", "bool", "string", "bytes":
					attr.Ty = work.Str
					for i := offset + 1; i < offset+10; i = i + 2 {
						if nl[i].Str == "." {
							attr.Ty += "." + nl[i+1].Str
							offset = i + 1
						} else {
							break
						}
					}
				}
			} else if attr.Name == "" {
				attr.Name = work.Str
			} else {
				attr.Num, _ = strconv.Atoi(work.Str)
				attr.Doc = doc(attr.Doc)
				got.Attr = append(got.Attr, attr)
				attr = Attr{}
			}
		default:
			attr.Doc += work.Str
		}
	}

	return got, newOffset
}

func protoOtherMessage(name string, l []*word) *Message {
	nl := l[1 : len(l)-1]
	got := Message{
		Name: name,
		Attr: make([]Attr, 0),
		Opt:  nil,
	}
	attr := Attr{}
	for offset := 0; offset < len(nl); offset++ {
		work := nl[offset]
		switch work.Ty {
		case wordT_word:
			if attr.Ty == "" {
				switch work.Str {
				case "message":
					attr.Ty = "message"
					fallthrough
				case "enum":
					attr.Ty = "enum"
					fallthrough
				case "oneof":
					if attr.Ty == "" {
						attr.Ty = "oneof"
					}
					attr.Name, _ = GetFistWord(nl[offset+1:])
					st, et := GetBrackets(nl[offset:], "{", "}")
					attr.Message = protoOtherMessage(attr.Name, nl[offset+st:offset+et+1])
					attr.Doc = doc(attr.Doc)
					got.Attr = append(got.Attr, attr)
					attr = Attr{}
					offset = offset + et + 1
				case "repeated": // 重复的
					attr.Repeated = true
				case "reserved": // 保留标识符
				default: // case "double", "float", "int32", "int64", "unit32", "unit64", "fixed32", "fixed64", "sfixed32", "sfixed64", "bool", "string", "bytes":
					attr.Ty = work.Str
					for i := offset + 1; i < offset+10; i = i + 2 {
						if nl[i].Str == "." {
							attr.Ty += "." + nl[i+1].Str
							offset = i + 1
						} else {
							break
						}
					}
				}
			} else if attr.Name == "" {
				attr.Name = work.Str
			} else {
				attr.Num, _ = strconv.Atoi(work.Str)
				attr.Doc = doc(attr.Doc)
				got.Attr = append(got.Attr, attr)
				attr = Attr{}
			}
		default:
			attr.Doc += work.Str
		}
	}

	return &got
}

func protoEnum(l []*word, offset int) (Enum, int) {
	name, i := GetFistWordBehindStr(l[offset:], "enum")
	offset = offset + i
	st, et := GetBrackets(l[offset:], "{", "}")
	newOffset := offset + et
	nl := l[offset+st+1 : offset+et]

	got := Enum{Name: name, Opt: make([]Attr, 0)}
	attr := Attr{}
	for offset := 0; offset < len(nl); offset++ {
		work := nl[offset]
		switch work.Ty {
		case wordT_word:
			if attr.Name == "" {
				attr.Name = work.Str
			} else {
				attr.Num, _ = strconv.Atoi(work.Str)
				attr.Doc = doc(attr.Doc)
				got.Opt = append(got.Opt, attr)
				attr = Attr{}
			}
		default:
			attr.Doc += work.Str
		}
	}

	return got, newOffset
}

func protoExtend(l []*word, offset int) (Message, int) {
	name, i := GetFistWordBehindStr(l[offset:], "extend")
	offset = offset + i
	st, et := GetBrackets(l[offset:], "{", "}")
	newOffset := offset + et
	nl := l[offset+st : offset+et]

	got := Message{
		Name: name,
		Attr: nil,
		Opt:  nil,
	}
	for offset := 0; offset < len(nl); offset++ {
		work := nl[offset]
		switch work.Ty {
		case wordT_line:
		case wordT_division:
		case wordT_doc:
		case wordT_word:

		}
	}

	return got, newOffset
}
