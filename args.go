package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Config struct {
	home    string
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

func parseArgs() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.home, "root", "", "指定のディレクトリへ/を割り当てファイルサーバーとします。指定がないとき/へのアクセスは404を返します。")
	flag.StringVar(&config.log, "log", "", "指定のパスにログファイルを出力します。指定がないときrp.logに出力します。")

	var reverseStrs stringSlice
	flag.Var(&reverseStrs,
		"reverse",
		"リバースプロキシを定義します。\n"+
			"\t\t--reverse aaa:1000:bbbと指定するとhttp://localhost/aaa/がhttp://localhost:1000/bbbに転送されます。\n"+
			"\t\t--reverse ccc:2000 のように指定するとhttp://localhost/cccがhttp://localhost:2000/ccc/に転送されます。")

	flag.Parse()

	log.Println(reverseStrs)

	if len(flag.Args()) > 0 {
		return nil, fmt.Errorf("too many arguments")
	}

	if config.log == "" {
		config.log = "rp.log"
	}

	for _, str := range reverseStrs {
		proxy, err := parseReverseProxies(str)
		if err != nil {
			return nil, fmt.Errorf("invalid reverse proxy format: %v", err)
		}
		config.reverse = append(config.reverse, proxy)
	}

	return config, nil
}

func parseReverseProxies(s string) (ReverseProxies, error) {
	var proxy ReverseProxies
	var err error

	args :=
		strings.Split(s, ":")
	if len(args) < 2 || len(args) > 3 {
		return proxy, fmt.Errorf("invalid format")
	}

	proxy.in_dir = args[0]
	proxy.port, err = strconv.Atoi(args[1])
	if err != nil {
		return proxy, err
	}
	if len(args) == 3 {
		proxy.out_dir = args[2]
	} else {
		proxy.out_dir = proxy.in_dir
	}

	return proxy, nil
}
