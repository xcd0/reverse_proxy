package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// リバースプロキシの設定を生成
func makeProxy(reqUrl string) http.Handler {

	rp := getReverseProxy(reqUrl)

	// Basic認証が不要なリバースプロキシとして振る舞うパス
	// reverse proxyが設定されているパスへのアクセスは別のポートに飛ばす
	outdir := "/"
	if rp.OutDir != "/" {
		outdir = fmt.Sprintf("/%s/", rp.OutDir)
	}
	indir := "/"
	if rp.InDir != "/" {
		indir = fmt.Sprintf("/%s/", rp.InDir)
	}
	out_url, _ := url.Parse(fmt.Sprintf("http://localhost:%d%s", rp.Port, outdir))
	proxy := httputil.NewSingleHostReverseProxy(out_url)
	// レスポンスの書き換えを行う
	proxy.ModifyResponse = func(response *http.Response) error {
		log.Printf("info: response rewrite : %v, response: %v", rp.rewriteContentsType, response)
		for i, t := range rp.rewriteContentsType {
			log.Printf("rp.rewriteContentsType.%v: %#v", i, t)
			if len(t) != 3 {
				log.Printf("rp.rewriteContentsType.%v: %#v : wrong format", i, t)
				continue
			}
			ext := t[0]
			preCType := t[1]
			postCType := t[2]
			ct := response.Header.Get("Content-Type")
			log.Printf("header contents-type: %#v", ct)
			//if len(ext) == 0 || strings.HasSuffix(response.Request.URL.Path, ext) { // URLの末尾が ext であればContent-Typeを ctype に設定 元のContent-Typeがtext/plainであるか確認
			//}
			if len(ct) == 0 || strings.Contains(ct, preCType) {
				ct_post := strings.ReplaceAll(ct, preCType, postCType)
				response.Header.Set("Content-Type", ct_post) // 条件に一致すればContent-Typeを変更
				log.Printf("Info: Rewrite contents type: ext:%v, %#v -> %#v", ext, ct, ct_post)
			}
			log.Printf("rewrited header contents-type: %#v", response.Header.Get("Content-Type"))
			response.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			response.Header.Set("Pragma", "no-cache")
			response.Header.Set("Expires", "0")
			log.Printf("rewrited header contents-type: %#v", response.Header.Get("Content-Type"))
		}
		return nil
	}
	log.Printf("in:%v,out:%v", indir, out_url)
	return proxy
}
