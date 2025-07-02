package analysis

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func ListHistoryFiles() {
	files, err := ioutil.ReadDir("history")
	if err != nil {
		fmt.Println("[历史记录] 无法读取 history 目录：", err)
		return
	}
	if len(files) == 0 {
		fmt.Println("[历史记录] 暂无历史分析记录。")
		return
	}
	fmt.Println("[历史记录] 可用分析记录：")
	for _, f := range files {
		if !f.IsDir() {
			fmt.Println(f.Name())
		}
	}
}

func ShowHistoryFile(filename string) {
	path := filepath.Join("history", filename)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("[历史记录] 读取失败：", err)
		return
	}
	fmt.Println(string(data))
}
