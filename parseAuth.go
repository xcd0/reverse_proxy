package main

import (
	"fmt"
	"log"
	"strings"
	"unsafe"

	"golang.org/x/crypto/bcrypt"
)

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
