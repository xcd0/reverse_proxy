# reverse proxy

ちょろっとURL転送したくてリバースプロキシほしいなというときに使う。
ロードバランサとかは考えていない。

## help

```sh
$ ./reverse_proxy --help
Usage of ./reverse_proxy:
  -auth value
        basic認証によるアクセス制限を設定します。
            --auth /aaa:alice:password のように指定して、http://localhost/aaa/へのアクセスをbasic認証でアクセス制限します。
            ディレクトリ指定は先頭に/をつけてください。
            --authの指定は複数指定できます。パスワードはhash化されて保持されます。再設定したい場合はサーバーを再起動させてください。
  -host string
        サーバーのドメインを指定します。指定がないときエラーです。
  -log string
        指定のパスにログファイルを出力します。指定がないときreverse_proxy.logに出力します。
  -reverse value
        リバースプロキシを定義します。
            --reverse aaa:1000:bbb のように指定するとhttp://localhost/aaa/がhttp://localhost:1000/bbbに転送されます。
            --reverse ccc:2000     のように指定するとhttp://localhost/ccc/がhttp://localhost:2000/ccc/に転送されます。
            --reverse ddd:3000:/   のように指定するとhttp://localhost/ddd/がhttp://localhost:3000/に転送されます。
  -root string
        指定のディレクトリへ/を割り当てファイルサーバーとします。指定がないとき/へのアクセスは404を返します。
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
	--root /var/www/html/ \
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
	--root /var/www/html/ \
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



