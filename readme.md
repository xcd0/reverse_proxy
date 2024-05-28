# reverse proxy

ちょろっとURL転送したくてリバースプロキシほしいなというときに使う。  
ロードバランサとかは考えていない。  

## 注意

リバースプロキシのお勉強用に書いているだけなので実用はお勧めしません。  
実用には [caddy](https://caddyserver.com/) をお勧めします。  

## コード整理中。

手探りで作っていたためスパゲッティ状態なので、コード整理しています。


## help

```sh
$ reverse_proxy --help
Usage of ./reverse_proxy:
  -auth value
        basic認証によるアクセス制限を設定します。
            --auth /aaa:alice:password のように指定して、http://localhost/aaa/へのアクセスをbasic認証でアクセス制限します。
            ディレクトリ指定は先頭に/をつけてください。
            --auth の指定は複数指定できます。パスワードはhash化されて保持されます。再設定したい場合は再起動してください。
  -host string
        サーバーのドメインを指定します。指定がないときエラーです。
  -log string
        指定のパスにログファイルを出力します。指定がないときreverse_proxy.logに出力します。
  -reverse value
        リバースプロキシを定義します。 --reverse の指定は複数指定できます。
        書式) <外部サブディレクトリ>:<ポート番号>:<内部サブディレクトリ>@<拡張子>:<元Contents-Type>:<新Contents-Type> @より後ろは複数可
            --reverse aaa:1000:bbb      のように指定すると http://localhost/aaa/  が http://localhost:1000/bbb  に転送されます。
            --reverse ccc:2000          のように指定すると http://localhost/ccc/  が http://localhost:2000/ccc/ に転送されます。
            --reverse ddd:3000:/        のように指定すると http://localhost/ddd/  が http://localhost:3000/     に転送されます。
            --reverse /:4000:eee        のように指定すると http://localhost/      が http://localhost:4000/eee  に転送されます。
            --reverse /:5000            のように指定すると http://localhost/      が http://localhost:5000/     に転送されます。
            --reverse /:f:/fuga         のように指定すると http://localhost/      を /fuga ディレクトリへのアクセスと見なし、ファイルサーバーとして振舞います。
            --reverse hoge:f:/fuga/piyo のように指定すると http://localhost/hoge/ を /fuga/piyoディレクトリへのアクセスと見なし、ファイルサーバーとして振舞います。
            --reverse ddd:3000:/@.html:text/plain:text/html のように指定すると http://localhost/ddd/ が http://localhost:3000/ に転送されつつ、
                URLの末尾が.htmlの時で、コンテンツタイプがtext/plainの場合コンテンツタイプをtext/htmlに書き換える、ようなことができます。
                @区切りで複数可能。'@.html:text/plain:text/html@.json:text/plain:application/json'
                拡張子は空にすると拡張子で制限しなくなる。例)'@:text/plain:text/html' 
  -vhost value
        name baseのvirtual host機能を提供します。 --vhost の指定は複数指定できます。
            --vhost aaa:/:80:/      のように指定して、 http://aaa.$host/     を http://localhost/      へ転送します。
            --vhost aaa:/:80:/dir   のように指定して、 http://aaa.$host/     を http://localhost/dir/  へ転送します。
            --vhost bbb:/:3000:/    のように指定して、 http://bbb.$host/     を http://localhost:3000/ へ転送します。
            --vhost bbb:/dir:4000:/ のように指定して、 http://bbb.$host/dir/ を http://localhost:4000/ へ転送します。
```

## 使用例

* サーバーのホスト名はexample.comとする。

* localhost:4000にgitlabのサーバーが立っている。
	* http://localhost/git/ ではgitlabのあるlocalhost:4000に転送したい。
* localhost:5000にredmineのサーバーが立っている。
	* http://localhost/redmine/ ではredmineのあるlocalhost:5000に転送したい。
* /var/www/html/index.htmlに静的なhtmlがおいてある。
	* http://localhost/へのアクセスには/var/www/html/index.htmlを返したい。
* /var/www/html/private/はユーザー名`user`パスワード`password`でアクセス権限したい。
	* http://localhost/private/へのアクセスはbasic認証を要求したい。

この時、  

```sh
$ ./reverse_proxy \
	--host example.com \
	--reverse /:f:/var/www/html/ \
	--reverse git:4000:/ \
	--reverse redmine:5000:/ \
	--auth /private:user:password
```

と実行する。

## FAQ

### http://localhost/git/ を http://localhost:4000/ に飛ばしたい。

--reverse git:4000:/ と指定する。

### http://localhost/git/ を http://localhost:4000/git/ に飛ばしたい。

--reverse git:4000:git と指定する。
--reverse git:4000 でもよい。

### http://localhost/git/ を http://localhost:4000/hogehoge/ に飛ばしたい。

--reverse git:4000:hogehoge と指定する。

### 複数アカウント許可したい。他のディレクトリも制限したい。

```
$ ./reverse_proxy \
	--host example.com \
	--reverse /:f:/var/www/html/ \
	--reverse git:4000:/ \
	--reverse redmine:5000:/ \
	--auth /private:user1:password1 \
	--auth /private:user2:password2 \
	--auth /private:user3:password3 \
	--auth /hoge/hoge:user1:password4 \
	--auth /piyo/piyo:user4:password5
```

のように複数指定する。
プログラムの内部ではパスワードはhash化して保持していて、生のパスワードはメモリ上にも保持していない。
アカウントやパスワードを変更したい場合はプログラムを再起動して再設定する。

現状している問題点は、呼び出しを自動化するためにはパスワードを自動化スクリプトに記述する必要があること。
この点は生のパスワードを記述しなくてよい仕組みが必要になる。


### aaa.example.com/hogeへのアクセスをexample.com:3000/piyo/に飛ばしたい。

virtual host機能を実装したものの、確認ができていない。

```
$ ./reverse_proxy \
	--host example.com \
	--vhost aaa:hoge:3000:piyo 
```

dns回りの設定ができていれば使えるはず。

