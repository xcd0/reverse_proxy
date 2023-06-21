package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// 引数reverseの後ろの文字列を解析する
// 例) --reverse aaa:999:bbb であればaaa:999:bbbの部分
// この場合サブディレクトリlocalhost/aaa/をlocalhost:999/bbb/に転送する
func parseReverseProxies(s string) (*ReverseProxies, error) {

	args := strings.Split(s, ":")
	log.Printf("in  : %v", s)
	log.Printf("args: %v", args)

	if len(args) < 2 || len(args) > 3 { // aaa:999:bbb or aaa:999
		return nil, fmt.Errorf("invalid format")
	}

	port := 80
	f := false
	if args[1] == "f" {
		f = true // ファイルサーバーとして振舞う
	} else {
		var err error
		port, err = strconv.Atoi(args[1])
		if err != nil {
			log.Printf("引数reverseによるポート番号の指定が不正です。: %s", args[1])
			return nil, err
		}
	}
	proxy := ReverseProxies{
		InDir:     args[0],
		Port:      port,
		OutDir:    args[0],
		FileServe: f,
	}
	if len(args) == 3 {
		proxy.OutDir = args[2]
	}
	return &proxy, nil
}
