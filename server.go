package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func generateServer(config *Config) {
	var writer io.Writer

	if config.log != "" {
		logFile, err := os.OpenFile(config.log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error opening log file: %v", err)
		}
		defer logFile.Close()
		writer = io.MultiWriter(os.Stderr, logFile)
	} else {
		writer = os.Stderr
	}
	log.SetOutput(writer)
	log.SetFlags(log.Ltime | log.Lshortfile)

	if config.home == "" {
		log.Println("homeの指定がありませんでした。/へのアクセスには404を返します。")
	} else {
		log.Printf("/へのアクセスには%vを返します。", config.home)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if config.home == "" {
			http.NotFound(w, r)
		} else {
			http.Redirect(w, r, config.home+r.URL.Path, http.StatusFound)
		}
	})

	if len(config.reverse) == 0 {
		log.Println("reverseの指定がありませんでした。")
	} else {
		for _, proxy := range config.reverse {
			if proxy.out_dir == "/" {
				log.Printf("/%s/ へのアクセスを http://localhost:%d/へ転送します。", proxy.in_dir, proxy.port)
			} else {
				log.Printf("/%s/ へのアクセスを http://localhost:%d/%s/へ転送します。", proxy.in_dir, proxy.port, proxy.out_dir)
			}
		}
	}

	// リバースプロキシの生成
	for _, rp := range config.reverse {
		proxyUrl, err := url.Parse(fmt.Sprintf("http://localhost:%d/", rp.port))
		if err != nil {
			log.Fatalf("invalid port number %d for reverse proxy %q: %v", rp.port, rp.in_dir, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(proxyUrl)

		http.HandleFunc(
			fmt.Sprintf("/%s/", rp.in_dir),
			func(w http.ResponseWriter, r *http.Request) {
				// httpsは対応していないかんじ
				// クライアントがリクエストしたURLを取得
				log.Println("---------------------------------------------")
				log.Printf("リクエストされたURL : %s", r.URL.String())

				/*
					originalURL := fmt.Sprintf("http://%s%s/", r.Host, r.RequestURI)
					targetURL, err := url.Parse(originalURL)
					if err != nil {
						log.Println(err)
						log.Printf("Error : 不正なURL : %s", originalURL)
					}
				*/
				log_request(r)

				log.Println("-- rewrite --")
				// example.com/aaa/にアクセスがあったとき
				// r.RequestURI と r.Pathの両方に/aaa/が入る
				// localhost:8080/aaa/に飛ばすとき
				// どうもそのままだとpath + requesturiしてしまうみたい

				originalURL := "http://" + r.Host + r.RequestURI
				target, _ := url.Parse(originalURL)
				r.URL.Host = target.Host
				r.URL.Scheme = target.Scheme
				r.URL.Path = target.Path

				r.URL.Host = r.Host // r.URL.HostにはリクエストされたURLを入れる
				//r.Host = fmt.Sprintf("localhost:%d", rp.port) // r.HostにプロキシサーバーのURLを入れる

				log_request(r)
				//log.Printf("%s/%s/", r.URL.Host, rp.in_dir)

				proxy.ServeHTTP(w, r)
			})
		/*
			http.Handle(
				fmt.Sprintf("/%s/", rp.in_dir),
				httputil.NewSingleHostReverseProxy(
					&url.URL{
						Scheme: "http",
						Host:   fmt.Sprintf("localhost:%d", rp.port),
					}),
			)
		*/
		if rp.out_dir == "/" {
			log.Printf("proxying http://localhost/%s/ to http://localhost:%d/", rp.in_dir, rp.port)
		} else {
			log.Printf("proxying http://localhost/%s/ to http://localhost:%d/%s/", rp.in_dir, rp.port, rp.out_dir)
		}
	}

	log.Fatal(http.ListenAndServe(":80", nil))
}

func log_request(r *http.Request) {
	u := r.URL
	log.Printf("r.Host: %s", r.Host)
	log.Printf("r.RequestURI : %s", r.RequestURI)
	log_url(u)
}
func log_url(u *url.URL) {
	log.Printf(
		"\nr.URL.Scheme : %v, \n r.URL.Opaque : %v, \n r.URL.User : %v, \n r.URL.Host : %v, \n r.URL.Path : %v, \n r.URL.RawPath : %v, \n r.URL.OmitHost : %v, \n r.URL.ForceQuery : %v, \n r.URL.RawQuery : %v, \n r.URL.Fragment : %v, \n r.URL.RawFragment : %v, \n ",
		u.Scheme,
		u.Opaque,
		u.User,
		u.Host,
		u.Path,
		u.RawPath,
		u.OmitHost,
		u.ForceQuery,
		u.RawQuery,
		u.Fragment,
		u.RawFragment,
	)
}
