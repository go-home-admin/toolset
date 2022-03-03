package parser

import (
	"bufio"
	"io/fs"
	"io/ioutil"
	"os"
	path2 "path"
)

type FileInfo struct {
	fs.FileInfo
	path string
}

// 读取目录中的所有文件包括子目录的文件
func loadFiles(path string, ext string) []FileInfo {
	got := make([]FileInfo, 0)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			t := loadFiles(path+"/"+file.Name(), ext)
			got = append(got, t...)
		} else if path2.Ext(file.Name()) == ext {
			got = append(got, FileInfo{
				FileInfo: file,
				path:     path + "/" + file.Name(),
			})
		}
	}

	return got
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
	// 引用中, "work" || 'k'
	scannerStatus_quote  scannerStatus = 6
	scannerStatus_quote2 scannerStatus = 7
	scannerStatus_quote3 scannerStatus = 8
)

// 对文件内容进行c系语言分词
func getWordsWitchFile(path string) GoWords {
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
			case scannerStatus_quote:
				work = work + str
				stop = true
				if str == "\"" && !HasSuffix(work, "\\\"") {
					got.list = append(got.list, &word{
						str: work,
						t:   wordT_word,
					})
					// 分割后从新开始
					work = ""
					status = scannerStatus_NewWork
				}
			case scannerStatus_quote2:
				work = work + str
				stop = true
				if str == "'" {
					got.list = append(got.list, &word{
						str: work,
						t:   wordT_word,
					})
					// 分割后从新开始
					work = ""
					status = scannerStatus_NewWork
				}
			case scannerStatus_quote3:
				work = work + str
				stop = true
				if str == "`" {
					got.list = append(got.list, &word{
						str: work,
						t:   wordT_word,
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
							got.list = append(got.list, &word{
								str: work,
								t:   wordT_word,
							})
						}
						got.list = append(got.list, &word{
							str: str,
							t:   wordT_division,
						})
						work = ""
						status = scannerStatus_NewWork
						lastIsSpe = true
					}
				} else {
					if len(work) != 0 {
						got.list = append(got.list, &word{
							str: work,
							t:   wordT_word,
						})
					}
					got.list = append(got.list, &word{
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
			if got.list[len(got.list)-1].t != wordT_line {
				got.list = append(got.list, &word{
					str: "\n",
					t:   wordT_line,
				})
			}
			status = scannerStatus_NewLine
			work = ""
		}
	}

	return got
}
