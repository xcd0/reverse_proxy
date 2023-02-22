# reverse proxy

ちょろっとURL転送したくてリバースプロキシほしいなというときに使う。
ロードバランサとかは考えていない。

## help

```sh
$ ./reverse_proxy --help
Usage of ./reverse_proxy:
  -log string
        指定のパスにログファイルを出力します。指定がないときrp.logに出力します。
  -reverse value
        リバースプロキシを定義します。
                --reverse aaa:1000:bbbと指定するとhttp://localhost/aaa/がhttp://localhost:1000/bbbに転送されます。
                --reverse ccc:2000 のように指定するとhttp://localhost/cccがhttp://localhost:2000/ccc/に転送されます。
  -root string
        指定のディレクトリへ/を割り当てファイルサーバーとします。指定がないとき/へのアクセスは404を返します。
```

## 使用例
仮に /var/www/html/ にindex.htmlが配置してあり、  
また4000番にdocker等でgitlabのサーバーが立っており、  
さらに5000番にredmineのサーバーが立っているという状況を考える。  
この時、 http://localhost/ ではindex.htmlが開き、  
http://localhost/git/ ではgitlabが開き、  
http://localhost/redmine/ ではredmineが開くという状態を作りたいとする。

この時、  
```sh
$ ./reverse_proxy --root /var/www/html/ --reverse git:4000 --reverse redmine:5000
```
とすると、  
http://localhost/ は /var/www/html/index.html を返し、  
http://localhost/git/ は http://localhost:4000/git/ を返し、  
http://localhost/redmine/ は http://localhost:5000/redmine/ を返すリバースプロキシサーバーが生成される。  

## FAQ

### http://localhost/git/ を http://localhost:4000/ に飛ばしたい。

--reverse git:4000:/ と指定する

### http://localhost/git/ を http://localhost:4000/hogehoge/ に飛ばしたい。

--reverse git:4000:hogehoge と指定する

