# reverse proxy

ちょろっとURL転送したくてリバースプロキシほしいなというときに使う。
ロードバランサとかは考えていない。

## help

```sh
$ ./reverse_proxy --help
Usage of ./reverse_proxy:
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
* /var/www/html/index.htmlに静的なhtmlがおいてある。
* localhost:4000にgitlabのサーバーが立っている。
* localhost:5000にredmineのサーバーが立っている。

という状況を考える。  

* http://localhost/へのアクセスには/var/www/html/index.htmlを返したい。
* http://localhost/git/ ではgitlabのあるlocalhost:4000に転送したい。
* http://localhost/redmine/ ではredmineのあるlocalhost:5000に転送したい。

この時、  

```sh
$ ./reverse_proxy --host example.com --root /var/www/html/ --reverse git:4000:/ --reverse redmine:5000:/
```

と実行する。

## FAQ

### http://localhost/git/ を http://localhost:4000/ に飛ばしたい。

--reverse git:4000:/ と指定する

### http://localhost/git/ を http://localhost:4000/git/ に飛ばしたい。

--reverse git:4000:git と指定する
--reverse git:4000 でもよい

### http://localhost/git/ を http://localhost:4000/hogehoge/ に飛ばしたい。

--reverse git:4000:hogehoge と指定する

