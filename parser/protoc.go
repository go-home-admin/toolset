package parser

import "fmt"

/**
golang parser 非完整token实现
*/
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
	Doc  string
	Name string
	ty   string
	num  int
}

type Enum struct {
	Doc  string
	Name string
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
				d.Doc = lastDoc
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
				val.Doc = lastDoc
				val, offset = protoOption(l.list, offset)
				d.Option[val.Key] = val
				lastDoc = ""
			case "service":
				var val Service
				val, offset = protoService(l.list, offset)
				val.Protoc = &d
				val.Doc = lastDoc
				d.Services[val.Name] = val
				lastDoc = ""
			case "message":
				var val Message
				val.Doc = lastDoc
				val, offset = protoMessage(l.list, offset)
				d.Messages[val.Name] = val
				lastDoc = ""
			case "enum":
				var val Enum
				val.Doc = lastDoc
				val, offset = protoEnum(l.list, offset)
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
func protoSyntax(l []*word, offset int) (string, int) {
	name, i := GetFistWordBehindStr(l[offset:], "syntax")
	return name[1 : len(name)-1], offset + i
}
func protoPackageName(l []*word, offset int) (string, int) {
	name, i := GetFistWordBehindStr(l[offset:], "package")
	return name, offset + i
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
				got.Opt[val.Key] = val
			case "rpc":
				var val ServiceRpc
				val, offset = protoRpc(nl, offset)
				got.Rpc[val.Name] = val
			}
		}
	}

	return got, newOffset
}

func protoRpc(l []*word, offset int) (ServiceRpc, int) {
	name, i := GetFistWordBehindStr(l[offset:], "rpc")
	offset = offset + i + 1
	Param, i := GetFistWord(l[offset:])
	offset = offset + i + 1
	Return, i := GetWord(l[offset:], 2)
	offset = offset + i + 1
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

func protoEnum(l []*word, offset int) (Enum, int) {
	name, i := GetFistWordBehindStr(l[offset:], "enum")
	offset = offset + i
	st, et := GetBrackets(l[offset:], "{", "}")
	newOffset := offset + et
	nl := l[offset+st : offset+et]

	got := Enum{
		Name: name,
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

func protoExtend(l []*word, offset int) (Message, int) {
	name, i := GetFistWordBehindStr(l[offset:], "enum")
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
