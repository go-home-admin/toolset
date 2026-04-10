package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/parser"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ProtocCommand @Bean
type ProtocCommand struct{}

func (ProtocCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:protoc",
		Description: "组装和执行protoc命令",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "proto",
					Description: "proto文件存放的目录",
					Default:     "@root/protobuf",
				},
				{
					Name:        "proto_path",
					Description: "protoc后面拼接的proto_path, 可以传入多个",
					Default:     "",
				},
				{
					Name:        "go_out",
					Description: "生成文件到指定目录",
					Default:     "@root/generate/proto",
				},
				{
					Name:        "out",
					Description: "其他扩展输出配置, 直接拼接值",
				},
			},
		},
	}
}

var show = false

// ctfang/command 对 "--foo=" 解析出的键为 "-foo"，与 Configure 里 Name 不一致时在此合并
func optionBoolTrue(input command.Input, name string) bool {
	if input.GetOption(name) == "true" {
		return true
	}
	return input.GetOption("-"+name) == "true"
}

func absPath(p string) string {
	a, err := filepath.Abs(filepath.Clean(p))
	if err != nil {
		return filepath.Clean(p)
	}
	return a
}

func protoPathArg(dir string) string {
	return "--proto_path=" + absPath(dir)
}

func (ProtocCommand) Execute(input command.Input) {
	// -debug=true / --debug=true：仅打印最终 protoc 命令（PrintCmd）
	show = optionBoolTrue(input, "debug")
	root := absPath(getRootPath())

	var outTemp, outPath string
	out := input.GetOption("go_out")
	out = strings.Replace(out, "@root", root, 1)
	if outIndex := strings.Index(out, ":"); outIndex != -1 && string(out[outIndex+1]) != "\\" {
		outPath = out[outIndex+1:]
		outPath = absPath(filepath.Join(outPath, "..", "temp"))
		_ = os.RemoveAll(outPath)
		_ = os.MkdirAll(outPath, 0766)
		outTemp = out[:outIndex+1] + outPath
		out = out[outIndex+1:]
	} else {
		outTemp = absPath(filepath.Join(filepath.Dir(out), "temp"))
		_ = os.RemoveAll(outTemp)
		_ = os.MkdirAll(outTemp, 0766)
		outPath = outTemp
	}

	path := input.GetOption("proto")
	path = strings.Replace(path, "@root", root, 1)
	path = absPath(path)
	out = absPath(out)
	outPath = absPath(outPath)
	outTemp = absPath(outTemp)

	pps := make([]string, 0)
	// 自动加入 protoc 安装目录下的 include，使 google/protobuf 等 well-known 优先于仓库内 node_modules 等误匹配
	if inc := protocBuiltinIncludeDir(root); inc != "" {
		inc = absPath(inc)
		pps = append(pps, protoPathArg(inc))
		for _, dir := range parser.GetChildrenDir(inc) {
			pps = append(pps, protoPathArg(dir.Path))
		}
	}
	// ctfang/command 对 "--proto_path=" 解析出的键为 "-proto_path"，与 Name 不一致时需合并
	protoPaths := append(append([]string(nil), input.GetOptions("proto_path")...), input.GetOptions("-proto_path")...)
	for _, s := range protoPaths {
		if s == "" {
			continue
		}
		s = absPath(strings.Replace(s, "@root", root, 1))
		pps = append(pps, protoPathArg(s))
		for _, dir := range parser.GetChildrenDir(s) {
			pps = append(pps, protoPathArg(dir.Path))
		}
	}
	// path/*.proto 不是protoc命令提供的, 如果这里执行需要每一个文件一个命令
	for _, dir := range parser.GetChildrenDir(path) {
		for _, info := range dir.GetFiles(".proto") {
			ppps := make([]string, len(pps))
			copy(ppps, pps)
			gof, _ := parser.GetProtoFileParser(info.Path)
			if gof.Imports != nil {
				for _, imts := range gof.Imports {
					imts = scanFileDir(root, imts)
					if imts != "" {
						ppps = append(ppps, protoPathArg(imts))
					}
				}
			}

			cods := []string{protoPathArg(dir.Path)}
			cods = append(cods, ppps...)
			cods = append(cods, "--go_out="+outTemp)
			cods = append(cods, absPath(info.Path))

			ProtocCmd(cods)
		}
	}

	// 生成后, 从temp目录拷贝到out
	_ = os.RemoveAll(out)
	rootAlias := filepath.ToSlash(pathTrimPrefixDir(out, root))
	module := filepath.ToSlash(getModModule())

	for _, dir := range parser.GetChildrenDir(outPath) {
		dir2 := filepath.ToSlash(pathTrimPrefixDir(dir.Path, outPath))
		dir3 := strings.TrimPrefix(dir2, module+"/")
		if dir2 == dir3 {
			continue
		}

		if dir3 == rootAlias {
			_ = os.Rename(dir.Path, out)
			_ = os.RemoveAll(outPath)
			break
		}
	}

	// 基础proto生成后, 生成Tag
	genProtoTag(out)
}

// pathTrimPrefixDir 从 full 中去掉 prefix 目录前缀（Windows / Unix 路径均兼容）
func pathTrimPrefixDir(full, prefix string) string {
	f := absPath(full)
	p := absPath(prefix)
	if f == p {
		return "."
	}
	rel, err := filepath.Rel(p, f)
	if err != nil || strings.HasPrefix(rel, "..") {
		s := strings.TrimPrefix(f, p)
		s = strings.TrimPrefix(s, string(filepath.Separator))
		s = strings.TrimPrefix(s, "/")
		return s
	}
	return rel
}

// resolveProtocExecutable 与 ProtocCmd 一致：优先项目 bin 下自带 protoc，否则 PATH 中的 protoc
func resolveProtocExecutable(root string) string {
	var rel string
	switch runtime.GOOS {
	case "darwin":
		rel = filepath.Join("bin", "protoc-mac")
	case "windows":
		rel = filepath.Join("bin", "protoc-win.exe")
	default:
		rel = filepath.Join("bin", "protoc-linux")
	}
	bundled := absPath(filepath.Join(root, rel))
	if st, err := os.Stat(bundled); err == nil && !st.IsDir() {
		return bundled
	}
	if p, err := exec.LookPath("protoc"); err == nil {
		return absPath(p)
	}
	return ""
}

// protocBuiltinIncludeDir 返回与将调用的 protoc 同级的 include（含 google/protobuf/descriptor.proto）
func protocBuiltinIncludeDir(root string) string {
	p := resolveProtocExecutable(root)
	if p == "" {
		return ""
	}
	if p2, err := filepath.EvalSymlinks(p); err == nil {
		p = p2
	}
	if p2, err := filepath.Abs(p); err == nil {
		p = p2
	}
	inc := filepath.Join(filepath.Dir(filepath.Dir(p)), "include")
	desc := filepath.Join(inc, filepath.FromSlash("google/protobuf/descriptor.proto"))
	if st, err := os.Stat(desc); err != nil || st.IsDir() {
		return ""
	}
	return inc
}

func skipImportSearchPath(dirPath string) bool {
	n := strings.ToLower(strings.ReplaceAll(dirPath, "\\", "/"))
	return strings.Contains(n, "/node_modules/") || strings.HasSuffix(n, "/node_modules") ||
		strings.Contains(n, "/vendor/") || strings.HasSuffix(n, "/vendor")
}

func scanFileDir(root, file string) string {
	imts := filepath.Join(root, filepath.FromSlash(file))
	if _, err := os.Stat(imts); !os.IsNotExist(err) {
		return root
	}

	for _, dir := range parser.GetChildrenDir(root) {
		if skipImportSearchPath(dir.Path) {
			continue
		}
		imts = filepath.Join(dir.Path, filepath.FromSlash(file))
		if _, err := os.Stat(imts); !os.IsNotExist(err) {
			return dir.Path
		}
	}
	return ""
}

func genProtoTag(out string) {
	byteLineDoc := []byte("//")
	bytePackage := []byte("package")
	byteT := []byte("\t")
	byteProtobuf := []byte("`protobuf:\"")
	byteProtobufOneof := []byte("`protobuf_oneof:\"")
	byteEq := []byte("=")

	for _, dir := range parser.GetChildrenDir(out) {
		for _, file := range dir.GetFiles(".go") {
			fd, err := os.Open(file.Path)
			defer fd.Close()
			if err != nil {
				fmt.Println("read error:", err)
			}

			fileNewStr := make([]byte, 0)
			buff := bufio.NewReader(fd)
			packageStart := false
			// 初始化文件Tag
			fileTags := make([]tag, 0)
			fileMapTags := map[string]tag{
				"form": {name: "Tag", key: "form", val: "{name}"},
			}
			for _, t := range fileMapTags {
				fileTags = append(fileTags, t)
			}
			// 开始替换文件内容
			lineTags := make([]tag, 0)
			lineMapTags := make(map[string]tag)
			for {
				data, _, eof := buff.ReadLine()
				if eof == io.EOF {
					break
				}
				newStr := append([]byte("\n"), data...)
				if bytes.Index(data, byteLineDoc) == -1 {
					if !packageStart {
						if bytes.Index(data, bytePackage) == 0 {
							packageStart = true
						} else {
							// 补充文件级别的 tag
							fileTags = append(fileTags, lineTags...)
							for s, t := range lineMapTags {
								fileMapTags[s] = t
							}
						}
					} else if bytes.HasPrefix(data, byteT) {
						var tagValue []byte
						end := 0
						if start := bytes.Index(data, byteProtobuf); start != -1 {
							nData := data[start:]
							begin := bytes.Index(nData, []byte("name="))
							end = bytes.Index(nData[begin:], []byte(","))
							nData = nData[begin : begin+end]
							tagValue = nData[bytes.Index(nData, byteEq)+1:]
							end = bytes.Index(data[start+begin:], []byte("`")) + start + begin
						} else if start := bytes.Index(data, byteProtobufOneof); start != -1 {
							nData := data[start:]
							begin := bytes.Index(nData, []byte("\"")) + 1
							end = bytes.Index(nData[begin:], []byte("\""))
							nData = nData[begin : begin+end]
							tagValue = nData[bytes.Index(nData, byteEq)+1:]
							end = bytes.Index(data[start+begin:], []byte("`")) + start + begin
						}
						if len(tagValue) != 0 {
							for _, t := range fileTags {
								if _, ok := lineMapTags[t.key]; !ok {
									lineMapTags[t.key] = t
									lineTags = append(lineTags, t)
								}
							}
							endStr := data[end:]
							newStr = append([]byte("\n"), data[:end]...)
							for _, lineTag := range lineTags {
								switch lineTag.key {
								case "json":
									newStr = []byte(strings.ReplaceAll(string(newStr), "json:\""+string(tagValue)+",omitempty\"", ""))
									newStr = append(newStr, []byte(" "+lineTag.key+":\""+
										strings.ReplaceAll(lineTag.val, "{name}", string(tagValue))+
										"\"")...)
								default:
									newStr = append(newStr, []byte(" "+lineTag.key+":\""+
										strings.ReplaceAll(lineTag.val, "{name}", string(tagValue))+
										"\"")...)
								}
							}
							newStr = append(newStr, endStr...)
						}
					}

					lineTags = make([]tag, 0)
					lineMapTags = make(map[string]tag)
				} else {
					str := strings.Trim(string(data), "")
					tags, mTags := getDocTag(str)
					lineTags = append(lineTags, tags...)
					for s, t := range mTags {
						lineMapTags[s] = t
					}
				}

				fileNewStr = append(fileNewStr, newStr...)
			}

			defer os.WriteFile(file.Path, []byte(strings.TrimSpace(string(fileNewStr))+"\n"), 0760)
		}
	}
}

type tag struct {
	name string

	key string
	val string
}

func getDocTag(doc string) ([]tag, map[string]tag) {
	got := make([]tag, 0)
	mapt := make(map[string]tag)

	arr := parser.GetWords(strings.ReplaceAll(doc, "//", " "))
	arrLet := len(arr)
	for i := 0; i < arrLet; i++ {
		w := arr[i]
		if w.Str == "@" {
			if arrLet < (i + 1) {
				continue
			}
			tag := tag{}
			tag.name = arr[i+1].Str
			if tag.name != "Tag" {
				continue
			}
			nl := arr[i+1:]
			i = i + 1
			st, et, has := parser.GetBracketsOrLn(nl, "(", ")")
			if !has {
				continue
			}
			i = i + et
			nl = nl[st+1 : et]
			for _, word := range nl {
				if word.Ty == 0 {
					if tag.key == "" {
						tag.key = word.Str[1 : len(word.Str)-1]
						tag.val = "{name}"
					} else {
						tag.val = word.Str[1 : len(word.Str)-1]
					}
				}
			}
			if tag.key != "" {
				got = append(got, tag)
				mapt[tag.key] = tag
			}
		}
	}

	return got, mapt
}

func ProtocCmd(params []string) {
	root := absPath(getRootPath())
	commandName := resolveProtocExecutable(root)
	if commandName == "" {
		commandName = "protoc"
		if _, err := exec.LookPath("protoc"); err != nil {
			log.Println("'protoc' 未安装; https://github.com/protocolbuffers/protobuf/releases")
			return
		}
	}

	// 打印真实命令
	if show {
		PrintCmd(commandName, params)
	}

	cmd := exec.Command(commandName, params...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}
}

func Cmd(commandName string, params []string) {
	// 打印真实命令
	if show {
		PrintCmd(commandName, params)
	}

	cmd := exec.Command(commandName, params...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalln(err)
	}
}

func PrintCmd(commandName string, params []string) {
	str := "\n" + commandName + " "
	for _, param := range params {
		str += param + " "
	}
	fmt.Print(str + "\n")
}
