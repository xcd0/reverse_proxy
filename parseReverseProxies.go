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
	{
		// 先頭に/が必ず1つつくようにする。。
		s = strings.TrimLeft(s, "/")
		s = "/" + s
		log.Printf("debug: s: %#v", s)
	}

	// @で区切る。
	args := strings.Split(s, "@")
	log.Printf("info: args: %v", args)

	// 1つ目はコロンで区切る。
	rpSettingArgs := strings.Split(args[0], ":")
	log.Printf("in  : %v", s)
	log.Printf("rpSettingArgs: %#v", rpSettingArgs)

	if len(rpSettingArgs) < 2 || len(rpSettingArgs) > 3 { // aaa:999:bbb or aaa:999
		return nil, fmt.Errorf("invalid format")
	}

	port := 80
	f := false
	if rpSettingArgs[1] == "f" {
		f = true // ファイルサーバーとして振舞う
	} else {
		var err error
		port, err = strconv.Atoi(rpSettingArgs[1])
		if err != nil {
			log.Printf("引数reverseによるポート番号の指定が不正です。: %s", rpSettingArgs[1])
			return nil, err
		}
	}

	proxy := ReverseProxies{
		InDir:               rpSettingArgs[0],
		Port:                port,
		OutDir:              rpSettingArgs[0],
		FileServe:           f,
		rewriteContentsType: [][]string{},
	}
	if len(rpSettingArgs) == 3 {
		proxy.OutDir = rpSettingArgs[2]
	}

	// コンテンツタイプの書き換え指定があれば書き換える。
	// "--reverse ddd:3000:/;.html:text/plain:text/html 対応
	// log.Printf("debug: args[1:]= %#v", args[1:])
	for _, contentsTypeMapStr := range args[1:] {
		log.Printf("debug: contentsTypeMapStr = %#v", contentsTypeMapStr)
		ctmap := strings.Split(contentsTypeMapStr, ":")
		log.Printf("debug: ctmap = %#v", ctmap)
		if len(ctmap) == 3 {
			log.Printf("Info: Rewrite contents type: ext:%v, %#v -> %#v", ctmap[0], ctmap[1], ctmap[2])
			proxy.rewriteContentsType = append(proxy.rewriteContentsType, ctmap)
		} else {
			// 書式間違い。
			log.Printf("Warning: %#v is wrong format. Skipped.", contentsTypeMapStr)
		}
	}
	log.Printf("info: proxy: %v", proxy)

	return &proxy, nil
}
