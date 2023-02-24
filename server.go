package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type Proxy struct {
	Targets  map[string]*httputil.ReverseProxy
	Handlers map[string]http.HandlerFunc
}

func create_rps(config *Config) *Proxy {

	proxy := Proxy{
		Targets:  map[string]*httputil.ReverseProxy{},
		Handlers: map[string]http.HandlerFunc{},
	}

	for _, rp := range config.reverse {
		u, err := url.Parse(fmt.Sprintf("http://localhost:%d/%s/", rp.port, rp.in_dir))
		if err != nil {
			log.Printf("error : ", fmt.Sprintf("http://%s:%d/%s/", config.host, rp.port, rp.in_dir))
			log.Fatal(err)
		}
		r := httputil.NewSingleHostReverseProxy(u)
		key := fmt.Sprintf("/%s", rp.in_dir)
		if r == nil {
			log.Printf("error : url %s nil", u)
			log.Printf("error : key %s nil", key)
			continue
		}
		proxy.Targets[key] = r
		proxy.Handlers[key] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//r.URL.Path = key + r.URL.Path
			proxy.Targets[key].ServeHTTP(w, r)
			log_request(r)
		})
	}

	return &proxy

}

func server(config *Config) {

	// 1段目のリバースプロキシサーバーの設定
	proxy := create_rps(config)

	mux := http.NewServeMux()
	//mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	fmt.Fprintf(w, "Hello World!")
	//})
	for _, rp := range config.reverse {
		key := fmt.Sprintf("/%s", rp.in_dir)
		mux.Handle(fmt.Sprintf("/%s/", rp.in_dir), proxy.Handlers[key])
		log.Printf("localhost/%s", key)
	}

	// リバースプロキシサーバーの起動
	log.Fatal(http.ListenAndServe(":80", mux))
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log_request(r)
	// 1段目のリバースプロキシサーバーにおいて、リクエストのパスに応じて
	// 2段目のリバースプロキシサーバーに転送するように条件分岐を設定
	for prefix, proxy := range p.Targets {
		if strings.HasPrefix(r.URL.Path, prefix) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			proxy.ServeHTTP(w, r)
			return
		}
	}

	// パスに一致する条件がない場合は404エラーを返す
	http.NotFound(w, r)
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
