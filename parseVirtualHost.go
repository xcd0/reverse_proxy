package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// 引数vhostの後ろの文字列を解析する
// 例) --vhost aaa:/:999:bbb であればaaa:/:999:bbbの部分
func parseVirtualHost(s string) (*VirtualHost, error) {
	args := strings.Split(s, ":")
	if len(args) != 4 { // aaa:/:999:bbb
		return nil, fmt.Errorf("invalid format")
	}
	port, err := strconv.Atoi(args[2])
	if err != nil {
		log.Printf("引数vhostによるポート番号の指定が不正です。: %s", args[2])
		return nil, err
	}
	vhost := VirtualHost{
		Vhost:  args[0],
		InDir:  args[1],
		Port:   port,
		OutDir: args[3],
	}
	return &vhost, nil
}
