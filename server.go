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
		if _, err := url.Parse(fmt.Sprintf("http://localhost:%d/", rp.port)); err != nil {
			log.Fatalf("invalid port number %d for reverse proxy %q: %v", rp.port, rp.in_dir, err)
		}
		http.Handle(
			fmt.Sprintf("/%s/", rp.in_dir),
			httputil.NewSingleHostReverseProxy(
				&url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("localhost:%d", rp.port),
				}),
		)
		if rp.out_dir == "/" {
			log.Printf("proxying http://localhost/%s/ to http://localhost:%d/", rp.in_dir, rp.port)
		} else {
			log.Printf("proxying http://localhost/%s/ to http://localhost:%d/%s/", rp.in_dir, rp.port, rp.out_dir)
		}
	}

	log.Fatal(http.ListenAndServe(":80", nil))
}
