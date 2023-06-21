package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"unsafe"

	"golang.org/x/crypto/bcrypt"
)

/*
// 基本はこれだけのコードで動く 他のコードは引数処理に過ぎない
func main() {
	gitURL, _ := url.Parse("http://localhost:8080/git/")
	http.Handle("/", http.FileServer(http.Dir("/mnt/d/public/html")))
	http.Handle("/git/", http.StripPrefix("/git/", httputil.NewSingleHostReverseProxy(gitURL)))
	http.ListenAndServe(":80", nil)
}
*/

// これはグローバルでないとダメ。httpHundleFuncから参照される。
var config *Config

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile) // ログの出力書式を設定する
	config = parseArgs()                     // 引数を解析しして構造体Configを作る
	log.Print(config)

	{
		router := http.NewServeMux()
		// reverse proxyが設定されていないパスへのアクセスは404を返す。
		for _, rp := range config.mapReverse {
			if rp.FileServe {
				// ディレクトリへのアクセスとみなす
				// ファイルサーバーとして振舞う。
				dir := "/"
				if rp.InDir != "/" {
					dir = fmt.Sprintf("/%s/", rp.InDir)
				}
				log.Printf("file serve : localhost:%d%v", rp.Port, dir)
				h := config.mapHandler[rp.InDir] // ファイルサーバーを実行する
				router.Handle(dir, http.StripPrefix(dir, h))
			} else {
				// リバースプロキシサーバーとして振舞う。
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
				router.Handle(indir, http.StripPrefix(indir, httputil.NewSingleHostReverseProxy(out_url)))
				log.Printf("in:%v,out:%v", indir, out_url)
			}
		}

		// virtual host
		for _, v := range config.vhost {
			out_url, _ := url.Parse(fmt.Sprintf("http://%s:%d/%s/", config.host, v.Port, v.OutDir))
			log.Printf("url : %s", out_url)
			log.Printf("in  : %s", v.InDir)
			log.Printf("out : %s", v.OutDir)
			log.Printf("port: %d", v.Port)
			in_url := ""
			if v.InDir == "/" {
				in_url = fmt.Sprintf("%s.%s/", v.Vhost, config.host)
			} else {
				in_url = fmt.Sprintf("%s.%s/%s/", v.Vhost, config.host, v.InDir)
			}
			router.Handle(
				in_url,
				http.StripPrefix(
					fmt.Sprintf("/%s/", v.InDir),
					httputil.NewSingleHostReverseProxy(out_url),
				),
			)
		}

		log.Fatal(http.ListenAndServe(":80", Log(router)))
	}
}

func Log(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rAddr := r.RemoteAddr
		method := r.Method
		path := r.URL.Path
		fmt.Printf("Remote: %s [%s] %s\n", rAddr, method, path)
		h.ServeHTTP(w, r)
	})
}

func isThisDirAccessControled(path string, config *Config) bool {
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
func authUser(w http.ResponseWriter, r *http.Request, config *Config) bool {
	username, password, ok := r.BasicAuth()
	defer zeroClear(&password)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	path := r.URL.Path

	// そもそもこのパスがアクセス禁止対象のパスかどうか
	//if !isThisDirAccessControled(path, config) {
	//	return true
	//}

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

// / *

func debug_request2(r *http.Request, r2 *http.Request) {
	log.Printf("r.Method           : %v\t->\t%v", r.Method, r2.Method)
	log.Printf("r.URL              : %v\t->\t%v", r.URL, r2.URL)
	log.Printf("r.Proto            : %v\t->\t%v", r.Proto, r2.Proto)
	log.Printf("r.ProtoMajor       : %v\t->\t%v", r.ProtoMajor, r2.ProtoMajor)
	log.Printf("r.ProtoMinor       : %v\t->\t%v", r.ProtoMinor, r2.ProtoMinor)
	log.Printf("r.Header           : %v\t->\t%v", r.Header, r2.Header)
	log.Printf("r.Body             : %v\t->\t%v", r.Body, r2.Body)
	log.Printf("r.GetBody          : %v\t->\t%v", r.GetBody, r2.GetBody)
	log.Printf("r.ContentLength    : %v\t->\t%v", r.ContentLength, r2.ContentLength)
	log.Printf("r.TransferEncoding : %v\t->\t%v", r.TransferEncoding, r2.TransferEncoding)
	log.Printf("r.Close            : %v\t->\t%v", r.Close, r2.Close)
	log.Printf("r.Host             : %v\t->\t%v", r.Host, r2.Host)
	log.Printf("r.Form             : %v\t->\t%v", r.Form, r2.Form)
	log.Printf("r.PostForm         : %v\t->\t%v", r.PostForm, r2.PostForm)
	log.Printf("r.MultipartForm    : %v\t->\t%v", r.MultipartForm, r2.MultipartForm)
	log.Printf("r.Trailer          : %v\t->\t%v", r.Trailer, r2.Trailer)
	log.Printf("r.RemoteAddr       : %v\t->\t%v", r.RemoteAddr, r2.RemoteAddr)
	log.Printf("r.RequestURI       : %v\t->\t%v", r.RequestURI, r2.RequestURI)
	log.Printf("r.TLS              : %v\t->\t%v", r.TLS, r2.TLS)
	log.Printf("r.Cancel           : %v\t->\t%v", r.Cancel, r2.Cancel)
	log.Printf("r.Response         : %v\t->\t%v", r.Response, r2.Response)

	debug_url2(r.URL, r2.URL)
}

func debug_url2(u *url.URL, u1 *url.URL) {
	log.Printf("r.URL.Scheme      : %v\t->\t%v", u.Scheme, u1.Scheme)
	log.Printf("r.URL.Opaque      : %v\t->\t%v", u.Opaque, u1.Opaque)
	log.Printf("r.URL.User        : %v\t->\t%v", u.User, u1.User)
	log.Printf("r.URL.Host        : %v\t->\t%v", u.Host, u1.Host)
	log.Printf("r.URL.Path        : %v\t->\t%v", u.Path, u1.Path)
	log.Printf("r.URL.RawPath     : %v\t->\t%v", u.RawPath, u1.RawPath)
	log.Printf("r.URL.OmitHost    : %v\t->\t%v", u.OmitHost, u1.OmitHost)
	log.Printf("r.URL.ForceQuery  : %v\t->\t%v", u.ForceQuery, u1.ForceQuery)
	log.Printf("r.URL.RawQuery    : %v\t->\t%v", u.RawQuery, u1.RawQuery)
	log.Printf("r.URL.Fragment    : %v\t->\t%v", u.Fragment, u1.Fragment)
	log.Printf("r.URL.RawFragment : %v\t->\t%v", u.RawFragment, u1.RawFragment)

}

func debug_request(r *http.Request) {
	log.Printf("r.Method           : %v", r.Method)
	log.Printf("r.URL              : %v", r.URL)
	log.Printf("r.Proto            : %v", r.Proto)
	log.Printf("r.ProtoMajor       : %v", r.ProtoMajor)
	log.Printf("r.ProtoMinor       : %v", r.ProtoMinor)
	log.Printf("r.Header           : %v", r.Header)
	log.Printf("r.Body             : %v", r.Body)
	log.Printf("r.GetBody          : %v", r.GetBody)
	log.Printf("r.ContentLength    : %v", r.ContentLength)
	log.Printf("r.TransferEncoding : %v", r.TransferEncoding)
	log.Printf("r.Close            : %v", r.Close)
	log.Printf("r.Host             : %v", r.Host)
	log.Printf("r.Form             : %v", r.Form)
	log.Printf("r.PostForm         : %v", r.PostForm)
	log.Printf("r.MultipartForm    : %v", r.MultipartForm)
	log.Printf("r.Trailer          : %v", r.Trailer)
	log.Printf("r.RemoteAddr       : %v", r.RemoteAddr)
	log.Printf("r.RequestURI       : %v", r.RequestURI)
	log.Printf("r.TLS              : %v", r.TLS)
	log.Printf("r.Cancel           : %v", r.Cancel)
	log.Printf("r.Response         : %v", r.Response)

	debug_url(r.URL)
}

func debug_url(u *url.URL) {
	log.Printf("r.URL.Scheme      : %v", u.Scheme)
	log.Printf("r.URL.Opaque      : %v", u.Opaque)
	log.Printf("r.URL.User        : %v", u.User)
	log.Printf("r.URL.Host        : %v", u.Host)
	log.Printf("r.URL.Path        : %v", u.Path)
	log.Printf("r.URL.RawPath     : %v", u.RawPath)
	log.Printf("r.URL.OmitHost    : %v", u.OmitHost)
	log.Printf("r.URL.ForceQuery  : %v", u.ForceQuery)
	log.Printf("r.URL.RawQuery    : %v", u.RawQuery)
	log.Printf("r.URL.Fragment    : %v", u.Fragment)
	log.Printf("r.URL.RawFragment : %v", u.RawFragment)

}

// * /
