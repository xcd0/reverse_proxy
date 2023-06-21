package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"unsafe"
)

// Config はアプリケーションの設定を保持するための構造体です。
// 引数で与えられた文字列を解析してConfigを生成します。
type Config struct {
	host    string
	vhost   []VirtualHost
	reverse []ReverseProxies
	auth    []Auth
	authDir map[string]bool // アクセス制限があるディレクトリ
	log     string

	mapReverse map[string]*ReverseProxies
	mapVhost   map[string]*VirtualHost
	mapHttpDir map[string]string
	mapHandler map[string]http.Handler
}

// ReverseProxies は引数で与えられた文字列から解析されたリバースプロキシの設定を保持する構造体です。
type ReverseProxies struct {
	Port      int
	InDir     string
	OutDir    string
	FileServe bool // ファイルサーバーとして振舞うか
}

type VirtualHost struct {
	Vhost  string
	Port   int
	InDir  string
	OutDir string
}

type Auth struct {
	Path     string
	UserName string
	Password []byte
}

// flag.Varで構造体に格納するためのインターフェースを満たすための記述
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func parseArgs() *Config {
	config := &Config{
		authDir:    map[string]bool{},
		mapReverse: map[string]*ReverseProxies{},
		mapVhost:   map[string]*VirtualHost{},
		mapHttpDir: map[string]string{},
		mapHandler: map[string]http.Handler{},
	}

	flag.StringVar(&config.host, "host", "", "サーバーのドメインを指定します。指定がないときエラーです。")
	flag.StringVar(&config.log, "log", "", "指定のパスにログファイルを出力します。指定がないときreverse_proxy.logに出力します。")
	var authOption stringSlice
	var reverseOption stringSlice
	var vhostOption stringSlice
	flag.Var(&reverseOption,
		"reverse",
		"リバースプロキシを定義します。 --reverse の指定は複数指定できます。\n"+
			"\t--reverse aaa:1000:bbb      のように指定すると http://localhost/aaa/  が http://localhost:1000/bbb  に転送されます。\n"+
			"\t--reverse ccc:2000          のように指定すると http://localhost/ccc/  が http://localhost:2000/ccc/ に転送されます。\n"+
			"\t--reverse ddd:3000:/        のように指定すると http://localhost/ddd/  が http://localhost:3000/     に転送されます。\n"+
			"\t--reverse /:4000:eee        のように指定すると http://localhost/      が http://localhost:4000/eee  に転送されます。\n"+
			"\t--reverse /:5000            のように指定すると http://localhost/      が http://localhost:5000/     に転送されます。\n"+
			"\t--reverse /:f:/fuga         のように指定すると http://localhost/      を /fuga ディレクトリへのアクセスと見なし、ファイルサーバーとして振舞います。\n"+
			"\t--reverse hoge:f:/fuga/piyo のように指定すると http://localhost/hoge/ を /fuga/piyoディレクトリへのアクセスと見なし、ファイルサーバーとして振舞います。",
	)
	flag.Var(&authOption,
		"auth",
		"basic認証によるアクセス制限を設定します。\n"+
			"\t--auth /aaa:alice:password のように指定して、http://localhost/aaa/へのアクセスをbasic認証でアクセス制限します。\n"+
			"\tディレクトリ指定は先頭に/をつけてください。\n"+
			"\t--auth の指定は複数指定できます。パスワードはhash化されて保持されます。再設定したい場合はサーバーを再起動させてください。",
	)
	flag.Var(&vhostOption,
		"vhost",
		"name baseのvirtual host機能を提供します。 --vhost の指定は複数指定できます。\n"+
			"\t--vhost aaa:/:80:/      のように指定して、 http://aaa.$host/     を http://localhost/      へ転送します。\n"+
			"\t--vhost aaa:/:80:/dir   のように指定して、 http://aaa.$host/     を http://localhost/dir/  へ転送します。\n"+
			"\t--vhost bbb:/:3000:/    のように指定して、 http://bbb.$host/     を http://localhost:3000/ へ転送します。\n"+
			"\t--vhost bbb:/dir:4000:/ のように指定して、 http://bbb.$host/dir/ を http://localhost:4000/ へ転送します。",
	)
	flag.Parse()

	// --log
	if config.log == "" {
		config.log = "reverse_proxy.log"
	}
	setLogfile(config.log) // logの設定をする

	// --host
	if config.host == "" {
		log.Println("host名を指定してください。")
		log.Fatal("例) --host example.com")
	}

	// --reverse
	for _, str := range reverseOption {
		proxy, err := parseReverseProxies(str) // 引数reverseの後ろの文字列を解析する
		if err != nil {
			log.Fatalf("invalid reverse proxy format: %v", err)
		}
		config.reverse = append(config.reverse, *proxy)
		config.mapReverse[proxy.InDir] = &config.reverse[len(config.reverse)-1]
		if proxy.FileServe {
			config.mapHttpDir[proxy.InDir] = proxy.OutDir
			config.mapHandler[proxy.InDir] = http.FileServer(http.Dir(proxy.OutDir))

		}
	}
	// reverseの指定で/の指定があればそこに飛ばす
	for k, v := range config.mapReverse {
		log.Printf("key : %v", k)
		log.Printf("val : %v", v)
	}

	if r, ok := config.mapReverse["/"]; ok {
		if r.Port == 80 {
			log.Printf("/へのアクセスには%vを返します。", r.OutDir)
		} else if r.OutDir == "/" {
			log.Printf("/へのアクセスには:%v/を返します。", r.Port)
		} else {
			log.Printf("/へのアクセスには:%v/%vを返します。", r.Port, r.OutDir)
		}
	} else {
		log.Println("/へのアクセスには404を返します。")
	}

	for i := 0; i < len(authOption); i++ {
		a, err := parseAuth(&authOption[i])
		if err != nil {
			log.Fatalf("invalid reverse proxy format: %v", err)
		}
		config.auth = append(config.auth, *a)
		config.authDir[a.Path] = true
	}
	for _, str := range vhostOption {
		vhost, err := parseVirtualHost(str) // 引数reverseの後ろの文字列を解析する
		if err != nil {
			log.Fatalf("invalid virtual host format: %v", err)
		}
		config.vhost = append(config.vhost, *vhost)
		config.mapVhost[str] = &config.vhost[len(config.vhost)-1]
	}
	defer func() {
		// メモリ上にパスワード絶対残さないマン
		for i := 0; i < len(authOption); i++ {
			zeroClear(&authOption[i])
		}
		for i := 1; i < len(os.Args); i++ {
			zeroClear(&os.Args[i])
		}
	}()
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

func zeroClear(s *string) {
	return
	b := unsafe.Slice(unsafe.StringData(*s), len(*s))
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}
