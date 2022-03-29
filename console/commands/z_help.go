package commands

import (
	"bufio"
	"os"
	"strings"
)

var rootPath string

func SetRootPath(root string) {
	rootPath = root
}

// 获取项目跟目录
func getRootPath() string {
	return rootPath
}

// 获取module
func getModModule() string {
	root := getRootPath()
	path := root + "/go.mod"

	fin, err := os.Stat(path)
	if err != nil {
		// no such file or dir
		panic("根目录必须存在go.mod")
	}
	if fin.IsDir() {
		panic("根目录必须存在go.mod文件")
	}

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	text := ""
	for scanner.Scan() {
		text = scanner.Text()
		break
	}
	text = strings.Replace(text, "module ", "", 1)
	return text
}
