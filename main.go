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
	runServer(config)     // サーバーを実行する
}

// 引数解析 {{{

// Config はアプリケーションの設定を保持するための構造体です。
// 引数で与えられた文字列を解析してConfigを生成します。
type Config struct {
	root    string
	host    string
	reverse []ReverseProxies
	log     string
}

// ReverseProxies は引数で与えられた文字列から解析されたリバースプロキシの設定を保持する構造体です。
type ReverseProxies struct {
	Port   int
	InDir  string
	OutDir string
}

// flag.Varで構造体に格納するためのインターフェースを満たすための記述  {{{2
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// }}}2

func parseArgs() *Config {
	config := &Config{}
	flag.StringVar(&config.host, "host", "", "サーバーのドメインを指定します。指定がないときエラーです。")
	flag.StringVar(&config.root, "root", "", "指定のディレクトリへ/を割り当てファイルサーバーとします。指定がないとき/へのアクセスは404を返します。")
	flag.StringVar(&config.log, "log", "", "指定のパスにログファイルを出力します。指定がないときreverse_proxy.logに出力します。")
	var reverseOption stringSlice
	flag.Var(&reverseOption,
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
	setLogfile(config.log) // logの設定をする
	for _, str := range reverseOption {
		proxy, err := parseReverseProxies(str) // 引数reverseの後ろの文字列を解析する
		if err != nil {
			log.Fatalf("invalid reverse proxy format: %v", err)
		}
		config.reverse = append(config.reverse, *proxy)
	}
	return config
}

func setLogfile(logfile string) {
	var writer io.Writer
	if logfile != "" {
		logFile, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("Error opening log file: %v", err)
		}
		defer logFile.Close()
		// 標準出力とログファイル両方にログを出力する
		writer = io.MultiWriter(os.Stderr, logFile)
	} else {
		writer = os.Stderr
	}
	log.SetOutput(writer)
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
		log.Printf("引数reverseによるポート番号の指定が不正です。: %s", args[1])
		return nil, err
	}
	proxy := ReverseProxies{
		InDir:  args[0],
		Port:   port,
		OutDir: args[0],
	}
	if len(args) == 3 {
		proxy.OutDir = args[2]
	}
	return &proxy, nil
}

// }}}

/*
// 基本はこれだけのコードで動く
func main() {
	gitURL, _ := url.Parse("http://localhost:8080/git/")
	http.Handle("/", http.FileServer(http.Dir("/mnt/d/public/html")))
	http.Handle("/git/", http.StripPrefix("/git/", httputil.NewSingleHostReverseProxy(gitURL)))
	http.ListenAndServe(":80", nil)
}
*/

func runServer(config *Config) {
	http.Handle("/", http.FileServer(http.Dir(config.root)))

	for _, r := range config.reverse {
		URL, _ := url.Parse(fmt.Sprintf("http://localhost:%d/%s/", r.Port, r.OutDir))
		log.Printf("url : %s", URL)
		log.Printf("in  : %s", r.InDir)
		log.Printf("out : %s", r.OutDir)
		log.Printf("port: %d", r.Port)
		http.Handle(fmt.Sprintf("/%s/", r.InDir), http.StripPrefix(fmt.Sprintf("/%s/", r.InDir), httputil.NewSingleHostReverseProxy(URL)))
	}

	log.Fatal(http.ListenAndServe(":80", nil))
}

/*
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
*/
