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
	"unsafe"

	"golang.org/x/crypto/bcrypt"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile) // ログの出力書式を設定する
}

/*
// 基本はこれだけのコードで動く 他のコードは引数処理に過ぎない
func main() {
	gitURL, _ := url.Parse("http://localhost:8080/git/")
	http.Handle("/", http.FileServer(http.Dir("/mnt/d/public/html")))
	http.Handle("/git/", http.StripPrefix("/git/", httputil.NewSingleHostReverseProxy(gitURL)))
	http.ListenAndServe(":80", nil)
}
*/

var config *Config

func main() {
	config = parseArgs() // 引数を解析しして構造体Configを作る
	log.Print(config)
	{
		// reverse proxyが設定されていないパスへのアクセスは
		// ディレクトリへのアクセスとみなす

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			debug_request(r)

			if dirAccessCheck(r.URL.Path) {
				if !authUser(w, r) { // ファイルへのアクセスを制限する
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			// ファイルサーバーを実行する
			http.FileServer(http.Dir(config.root)).ServeHTTP(w, r)
		})

		// reverse proxyが設定されているパスへのアクセスは別のポートに飛ばす
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
}

func dirAccessCheck(path string) bool {
	/*
		if path == "/" {
			return true // 全てアクセス制限
		}
	*/

	// アクセスしようとしているpathが /a/b/cで
	paths := strings.Split(path, "/") // a, b, c
	tmp := "/"
	log.Println("path : ", path)
	log.Println("conf : ", config.authDir)

	for _, d := range paths {
		tmp, _ = url.JoinPath(tmp, d) // a, a/b, a/b/cという感じに調べる
		v, ok := config.authDir[tmp]
		log.Printf("check dir : %s, %v, %v, d : %v", tmp, v, ok, d)
		if ok {
			// アクセス禁止対象ディレクトリであった
			return true
		}
	}
	return false
}

// ベーシック認証を実行するミドルウェア
func authUser(w http.ResponseWriter, r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	defer zeroClear(&password)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	path := r.URL.Path

	// そもそもこのパスがアクセス禁止対象のパスかどうか
	prohibit := dirAccessCheck(path)
	if !prohibit {
		return true
	}

	for _, a := range config.auth {
		log.Println("path : ", path)
		log.Println("apath : ", a.Path)
		log.Println("index : ", strings.Index(path, a.Path))
		dirok := 0 == strings.Index(path, a.Path)
		nameok := string(username) == a.UserName
		passok := func() bool {
			if err := bcrypt.CompareHashAndPassword(a.Password, unsafe.Slice(unsafe.StringData(password), len(password))); err != nil {
				return false
			}
			return true
		}()

		log.Printf("a %v", a)
		log.Printf("path %v, aPath %v", path, a.Path)
		log.Printf("username %s, aUserName %s", string(username), a.UserName)
		log.Printf("hashed %s, aPasswd %s", string(a.Password), string(password))
		log.Printf("dir %v, name %v, pass %v", dirok, nameok, passok)

		if dirok && nameok && passok {
			return true
		}
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return false
}

// 引数解析 {{{

// Config はアプリケーションの設定を保持するための構造体です。
// 引数で与えられた文字列を解析してConfigを生成します。
type Config struct {
	root    string
	host    string
	reverse []ReverseProxies
	auth    []Auth
	authDir map[string]bool // アクセス制限があるディレクトリ
	log     string
}

// ReverseProxies は引数で与えられた文字列から解析されたリバースプロキシの設定を保持する構造体です。
type ReverseProxies struct {
	Port   int
	InDir  string
	OutDir string
}

type Auth struct {
	Path     string
	UserName string
	Password []byte
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
	config := &Config{authDir: map[string]bool{}}
	flag.StringVar(&config.host, "host", "", "サーバーのドメインを指定します。指定がないときエラーです。")
	flag.StringVar(&config.root, "root", "", "指定のディレクトリへ/を割り当てファイルサーバーとします。指定がないとき/へのアクセスは404を返します。")
	flag.StringVar(&config.log, "log", "", "指定のパスにログファイルを出力します。指定がないときreverse_proxy.logに出力します。")
	var authOption stringSlice
	var reverseOption stringSlice
	flag.Var(&reverseOption,
		"reverse",
		"リバースプロキシを定義します。\n"+
			"\t--reverse aaa:1000:bbb のように指定するとhttp://localhost/aaa/がhttp://localhost:1000/bbbに転送されます。\n"+
			"\t--reverse ccc:2000     のように指定するとhttp://localhost/ccc/がhttp://localhost:2000/ccc/に転送されます。\n"+
			"\t--reverse ddd:3000:/   のように指定するとhttp://localhost/ddd/がhttp://localhost:3000/に転送されます。",
	)
	flag.Var(&authOption,
		"auth",
		"basic認証によるアクセス制限を設定します。\n"+
			"\t--auth /aaa:alice:password のように指定して、http://localhost/aaa/へのアクセスをbasic認証でアクセス制限します。\n"+
			"\tディレクトリ指定は先頭に/をつけてください。\n"+
			"\t--authの指定は複数指定できます。パスワードはhash化されて保持されます。再設定したい場合はサーバーを再起動させてください。",
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
	for i := 0; i < len(authOption); i++ {
		a, err := parseAuth(&authOption[i])
		if err != nil {
			log.Fatalf("invalid reverse proxy format: %v", err)
		}
		config.auth = append(config.auth, *a)
		config.authDir[a.Path] = true
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

func zeroClear(s *string) {
	return
	b := unsafe.Slice(unsafe.StringData(*s), len(*s))
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}

// 引数authの後ろの文字列を解析する
// 例) --auth /dir:bob:passwd であれば /dir:bob:passwd
func parseAuth(s *string) (*Auth, error) {
	if 0 != strings.Index(*s, "/") { // 最初の1文字目はスラッシュ
		zeroClear(s)
		fmt.Println("Error : アクセス制限するディレクトリ指定は先頭にスラッシュをつけてください。")
		fmt.Println("        例) --auth /dir:taro:imo ")
		return nil, fmt.Errorf("invalid format")
	}
	args := strings.Split(*s, ":")
	if len(args) != 3 { // aaa:999:bbb or aaa:999
		zeroClear(s)
		return nil, fmt.Errorf("invalid format")
	}
	hashed, err := bcrypt.GenerateFromPassword(
		unsafe.Slice(unsafe.StringData(args[2]), len(args[2])),
		bcrypt.DefaultCost)
	log.Println(args[1])
	//log.Println(string(hashed))

	if err := bcrypt.CompareHashAndPassword(hashed, []byte(args[2])); err != nil {
		log.Println(err)
	}

	if err != nil {
		return nil, err
	}
	a := Auth{
		Path:     args[0],
		UserName: args[1],
		Password: hashed,
	}
	zeroClear(s)
	zeroClear(&args[2])
	return &a, nil
}

// }}}

// / *
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

// * /
