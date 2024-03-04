package main

import (
	"fmt"
	"log"
	"net/http"
)

// ファイルサーバーの設定を生成
func makeFileServer(url string) http.Handler {

	rp := getReverseProxy(url)

	// Basic認証が不要なファイルサーバとして振る舞うパス
	dir := "/"
	if rp.InDir != "/" {
		dir = fmt.Sprintf("/%s/", rp.InDir)
	}
	log.Printf("file serve : localhost:%d%v", rp.Port, dir)

	return http.StripPrefix(dir, config.mapHandler[rp.InDir])

	// すでに以下のように設定されている
	// config.mapHttpDir[proxy.InDir] = proxy.OutDir
	// config.mapHandler[proxy.InDir] = http.FileServer(http.Dir(proxy.OutDir))
}
