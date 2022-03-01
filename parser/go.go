package parser

import (
	"bufio"
	"fmt"
	"os"
)

type DirParser struct {
	PackageName string
	Imports     map[string]string
	Types       map[string]string
	Funcs       map[string]string
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
	l := words(info.path)
	for _, i := range l.list {
		fmt.Print(i.str)
	}

	return "", nil
}

// 分词后的结构
type word struct {
	str string
	t   wordT
}
type GoWords struct {
	list []*word
}
type wordT int

const (
	// 单词
	wordT_word wordT = 0
	// 分隔符
	wordT_division wordT = 1
	// 换行符
	wordT_line wordT = 2
	wordT_doc  wordT = 3
)

// 分词过程状态
type scannerStatus int

const (
	// 新的一行
	scannerStatus_NewLine scannerStatus = 0
	// 准备注释中
	scannerStatus_DocWait scannerStatus = 1
	// 注释中, 单行, 多行
	scannerStatus_Doc  scannerStatus = 2
	scannerStatus_Doc2 scannerStatus = 3
	// 遇到间隔符号
	scannerStatus_NewWork scannerStatus = 4
	// 单词中
	scannerStatus_Work scannerStatus = 5
)

func words(path string) GoWords {
	var got = GoWords{
		list: make([]*word, 0),
	}
	file, err := os.Open(path)
	if err != nil {
		panic(err)
		return got
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	status := scannerStatus_NewLine
	work := ""
	lastIsSpe := false
	for scanner.Scan() {
		for _, s := range scanner.Text() {
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
					got.list = append(got.list, &word{
						str: work,
						t:   wordT_doc,
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
					got.list = append(got.list, &word{
						str: work,
						t:   wordT_division,
					})
					// 分割后从新开始
					work = ""
					status = scannerStatus_NewWork
				}
			case scannerStatus_NewLine, scannerStatus_NewWork:
				if str == "/" {
					work = work + str
					status = scannerStatus_DocWait
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
						got.list = append(got.list, &word{
							str: work,
							t:   wordT_division,
						}, &word{
							str: str,
							t:   wordT_division,
						})
						work = ""
						status = scannerStatus_NewWork
						lastIsSpe = true
					}
				} else {
					got.list = append(got.list, &word{
						str: work,
						t:   wordT_word,
					}, &word{
						str: str,
						t:   wordT_division,
					})
					work = ""
					status = scannerStatus_NewWork
					lastIsSpe = false
				}
			} else {
				lastIsSpe = false
			}
		}
		switch status {
		case scannerStatus_Work:
			got.list = append(got.list, &word{
				str: work,
				t:   wordT_word,
			}, &word{
				str: "\n",
				t:   wordT_line,
			})
			status = scannerStatus_NewLine
			work = ""
		case scannerStatus_Doc:
			got.list = append(got.list, &word{
				str: work,
				t:   wordT_doc,
			}, &word{
				str: "\n",
				t:   wordT_line,
			})
			status = scannerStatus_NewLine
			work = ""
		case scannerStatus_Doc2:
			// 多行注释未结束
			work = work + "\n"
		default:
			got.list = append(got.list, &word{
				str: "\n",
				t:   wordT_line,
			})
			status = scannerStatus_NewLine
			work = ""
		}
	}

	return got
}
