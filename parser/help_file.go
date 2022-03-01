package parser

import (
	"io/fs"
	"io/ioutil"
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
