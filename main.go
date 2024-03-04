package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var handler http.Handler

		// リクエストのパスに応じてハンドラを設定
		if IsFileServe(r.URL.Path) {
			handler = makeFileServer(r.URL.Path)
		} else {
			handler = makeProxy(r.URL.Path)
		}

		log.Printf("config.authDirs : %#v", config.authDirs) // アクセス制限があるディレクトリ

		// config.authに基づいてBasic認証を適用
	BASIC_AUTH_SEARCH:
		for _, ad := range config.authDirs {
			log.Printf("url: %#v >< %#v", r.URL.Path, ad)
			if strings.HasPrefix(r.URL.Path, ad) {
				log.Printf("url: %#v ⊃  %#v", r.URL.Path, ad)
				for _, authConfig := range config.auth {
					if authConfig.Path == ad {
						log.Printf("find authConfig : %#v", authConfig)
						handler = BasicAuth(handler, &authConfig)
						break BASIC_AUTH_SEARCH
					}
				}
				log.Printf("find authConfig : NG")
			}
		}
		handler.ServeHTTP(w, r)
	})

	http.ListenAndServe(":80", nil)
}

func IsFileServe(path string) bool {
	for p, rp := range config.mapReverse {
		if strings.HasPrefix(path, p) {
			if rp.FileServe {
				return true
			}
		}
	}
	return false
}

func contains(a string, b []string) bool {
	for _, v := range b {
		if a == v {
			return true // a が b の中に見つかった場合、true を返す
		}
	}
	return false // b の中に a が見つからなかった場合、false を返す
}

/*
func oldmain() {
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
				router.Handle(dir, http.StripPrefix(func(rp *ReverseProxies) string {
					dir := "/"
					if rp.InDir != "/" {
						dir = fmt.Sprintf("/%s/", rp.InDir)
					}
					log.Printf("file serve : localhost:%d%v", rp.Port, dir)
				}(rp), config.mapHandler[rp.InDir]))
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

func CheckBasicAuth(dir string) []Auth {
	auth := []Auth{}
	for _, a := range config.authDirs {
		sa := string(a)
		if strings.HasPrefix(dir, sa) { // outdirもaも/で始まる。
			log.Printf("check dir : %s >< %v", dir, sa)
			// アクセス制限対象ディレクトリであった
			log.Printf("%v is protected by basic authentication.", dir)
			for i, ca := range config.auth {
				log.Printf("ca.Path: %#v, a: %#v", ca.Path, sa)
				if ca.Path == sa {
					auth = append(auth, config.auth[i]) // 以降の処理で使いやすいようにauthに入れておく。
					log.Printf("path:%#v, user:%#v, hashed password:%#v", config.auth[i].Path, config.auth[i].UserName, string(config.auth[i].Password))
				}
			}
			if len(auth) == 0 {
				// 謎
				log.Fatalf("バグ : マッチするはずのところでスルーした")
			}
		}
	}
	return auth
}

*/

func Log(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rAddr := r.RemoteAddr
		method := r.Method
		path := r.URL.Path
		fmt.Printf("Remote: %s [%s] %s\n", rAddr, method, path)
		h.ServeHTTP(w, r)
	})
}

/*
func isThisDirAccessControled(path string, config *Config) bool {
//		if path == "/" {
//			return true // 全てアクセス制限
//		}

	// アクセスしようとしているpathが /a/b/cで
	paths := strings.Split(path, "/") // a, b, c
	log.Println("path : ", path)
	log.Println("conf : ", config.authDirs)

	for _, d := range paths {
		tmp := "/"
		tmp, _ = url.JoinPath(tmp, d) // a, a/b, a/b/cという感じに調べる
		for _, a := range config.authDirs {
			log.Printf("check dir : %s >< %v", tmp, a)
			if strings.HasPrefix(tmp, a) {
				// アクセス禁止対象ディレクトリであった
				log.Printf("%v is protected by basic authentication.", tmp)
				return true
			}
			log.Printf("check ok: %s", tmp)
		}
	}
	return false
}

// ベーシック認証を実行するミドルウェア
func authUser(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	defer zeroClear(&password)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	proxy.ServeHTTP(w, r)
	return
	//////////////

	path := r.URL.Path

	// そもそもこのパスがアクセス禁止対象のパスかどうか
	//if !isThisDirAccessControled(path, config) {
	//	return
	//}

	for _, a := range config.auth {
		log.Println("path : ", path)
		log.Println("apath : ", a.Path)
		log.Println("index : ", strings.Index(path, a.Path))
		dirok := 0 == strings.Index(path, a.Path)
		nameok := string(username) == a.UserName
		passok := func() bool {
			if err := bcrypt.CompareHashAndPassword(a.Password, unsafe.Slice(unsafe.StringData(password), len(password))); err != nil {
				return
			}
			return
		}()

		log.Printf("a %v", a)
		log.Printf("path %v, aPath %v", path, a.Path)
		log.Printf("username %s, aUserName %s", string(username), a.UserName)
		log.Printf("hashed %s, aPasswd %s", string(a.Password), string(password))
		log.Printf("dir %v, name %v, pass %v", dirok, nameok, passok)

		if dirok && nameok && passok {
			return
		}
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return
}

*/

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
