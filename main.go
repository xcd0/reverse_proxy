package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile) // ログの出力書式を設定する
}

func main() {
	config := parseArgs() // 引数を解析しして構造体Configを作る
	RunServer(config)     // サーバーを実行する
}

// 引数解析 {{{
type Config struct {
	root    string
	host    string
	reverse []ReverseProxies
	log     string
}

type ReverseProxies struct {
	port    int
	in_dir  string
	out_dir string
}

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func parseArgs() *Config {
	config := &Config{}
	flag.StringVar(&config.host, "host", "", "サーバーのドメインを指定します。指定がないときエラーです。")
	flag.StringVar(&config.root, "root", "", "指定のディレクトリへ/を割り当てファイルサーバーとします。指定がないとき/へのアクセスは404を返します。")
	flag.StringVar(&config.log, "log", "", "指定のパスにログファイルを出力します。指定がないときreverse_proxy.logに出力します。")
	var reverseStrs stringSlice
	flag.Var(&reverseStrs,
		"reverse",
		"リバースプロキシを定義します。\n"+
			"\t--reverse aaa:1000:bbb のように指定するとhttp://localhost/aaa/がhttp://localhost:1000/bbbに転送されます。\n"+
			"\t--reverse ccc:2000     のように指定するとhttp://localhost/ccc/がhttp://localhost:2000/ccc/に転送されます。\n"+
			"\t--reverse ddd:3000:/   のように指定するとhttp://localhost/ddd/がhttp://localhost:3000/に転送されます。",
	)
	flag.Parse()

	if config.host == "" {
		log.Println("host名を指定してください。")
		log.Fatal("例) --host example.com")
	}
	if config.root == "" {
		log.Println("引数rootの指定がありませんでした。/へのアクセスには404を返します。")
	} else {
		log.Printf("/へのアクセスには%vを返します。", config.root)
	}
	if config.log == "" {
		config.log = "reverse_proxy.log"
	}
	SetLogfile(config.log) // logの設定をする
	for _, str := range reverseStrs {
		proxy, err := parseReverseProxies(str) // 引数reverseの後ろの文字列を解析する
		if err != nil {
			log.Fatalf("invalid reverse proxy format: %v", err)
		}
		config.reverse = append(config.reverse, *proxy)
	}
	return config
}

func SetLogfile(logfile string) { // 標準出力とログファイル両方にログを出力する
	var writer io.Writer
	if logfile != "" {
		logFile, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
}

// 引数reverseの後ろの文字列を解析する
// 例) --reverse aaa:999:bbb であればaaa:999:bbbの部分
// この場合サブディレクトリlocalhost/aaa/をlocalhost:999/bbb/に転送する
func parseReverseProxies(s string) (*ReverseProxies, error) {
	args := strings.Split(s, ":")
	if len(args) < 2 || len(args) > 3 { // aaa:999:bbb or aaa:999
		return nil, fmt.Errorf("invalid format")
	}
	port, err := strconv.Atoi(args[1])
	if err != nil {
		log.Printf("引数reverseによるポート番号の指定が不正です。", args[1])
		return nil, err
	}
	proxy := ReverseProxies{
		in_dir:  args[0],
		port:    port,
		out_dir: args[0],
	}
	if len(args) == 3 {
		proxy.out_dir = args[2]
	}
	return &proxy, nil
}

// }}}

// サーバー実行 {{{
func RunServer(config *Config) {
	mux := http.NewServeMux()   // 1段目のプロキシサーバー ホスト名の書き換えとルーティング
	proxy := create_rps(config) // 2段目のリバースプロキシサーバーの設定
	for _, rp := range config.reverse {
		mux.Handle(
			fmt.Sprintf("/%s/", rp.in_dir),
			proxy.Handlers[fmt.Sprintf("/%s", rp.in_dir)],
		)

	}
	log.Fatal(http.ListenAndServe(":80", mux)) // リバースプロキシサーバーの起動
}

// 2段目のリバースプロキシサーバーを指定されたディレクトリ分作る
func create_rps(config *Config) *Proxy {
	proxy := Proxy{
		Targets:  map[string]*httputil.ReverseProxy{},
		Handlers: map[string]http.HandlerFunc{},
	}
	for _, rp := range config.reverse {
		// 転送先のURLのチェック
		u, err := url.Parse(fmt.Sprintf("http://localhost:%d/%s/", rp.port, rp.in_dir))
		if err != nil {
			log.Printf("error : ", fmt.Sprintf("http://%s:%d/%s/", config.host, rp.port, rp.in_dir))
			log.Fatal(err)
		}
		r := httputil.NewSingleHostReverseProxy(u) // 転送先のURLに転送させるリバースプロキシサーバー
		key := fmt.Sprintf("/%s", rp.in_dir)       // 構造体Proxyのmapのキー
		if r == nil {
			log.Printf("error : Cannot create reverse proxy server. url : %s. key : %s ", u, key)
			continue
		}
		proxy.Targets[key] = r
		proxy.Handlers[key] = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//r.URL.Path = key + r.URL.Path
			proxy.Targets[key].ServeHTTP(w, r)
			debug_request(r)
		})
	}
	return &proxy
}

type Proxy struct {
	// mapのkeyは/aaaのように最初の1字は/とする
	Targets  map[string]*httputil.ReverseProxy
	Handlers map[string]http.HandlerFunc
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debug_request(r)
	// 1段目のリバースプロキシサーバーにおいて、リクエストのパスに応じて
	// 2段目のリバースプロキシサーバーに転送するように条件分岐を設定
	for prefix, proxy := range p.Targets {
		if strings.HasPrefix(r.URL.Path, prefix) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			proxy.ServeHTTP(w, r)
			return
		}
	}
	log.Printf("Warning : status 404 : %s%s", r.Host, r.RequestURI)
	http.NotFound(w, r) // パスに一致する条件がない場合は404エラーを返す
}

func debug_request(r *http.Request) {
	u := r.URL
	log.Printf("r.Host: %s", r.Host)
	log.Printf("r.RequestURI : %s", r.RequestURI)
	debug_url(u)
}

func debug_url(u *url.URL) {
	log.Printf(
		"\nr.URL.Scheme : %v, \n r.URL.Opaque : %v, \n r.URL.User : %v, \n r.URL.Host : %v, \n r.URL.Path : %v, \n r.URL.RawPath : %v, \n r.URL.OmitHost : %v, \n r.URL.ForceQuery : %v, \n r.URL.RawQuery : %v, \n r.URL.Fragment : %v, \n r.URL.RawFragment : %v, \n ",
		u.Scheme, u.Opaque, u.User, u.Host, u.Path, u.RawPath, u.OmitHost, u.ForceQuery, u.RawQuery, u.Fragment, u.RawFragment)
}

// }}}
