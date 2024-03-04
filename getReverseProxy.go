package main

import (
	"log"
	"strings"
)

func getReverseProxy(url string) *ReverseProxies {

	log.Printf("url:%#v", url)

	matched := []*ReverseProxies{}

	tmp := ""
	for map_path, _ := range config.mapReverse {
		tmp += map_path + ", "
	}
	log.Printf("config.mapReverse : %#v", tmp)

	var rp *ReverseProxies = nil
	for map_path, map_rp := range config.mapReverse {
		if strings.HasPrefix(url, map_path) {
			matched = append(matched, map_rp)
		}
	}

	log.Printf("matched : %#v", matched)

	if len(matched) != 0 {
		rp = matched[0]
		for i := 1; i < len(matched); i++ {
			rpi := matched[i]
			if len(rp.InDir) < len(rpi.InDir) {
				rp = rpi
			}
		}
	}

	if rp == nil {
		log.Printf("Error: rp is nil")
		return nil
	}

	log.Printf("rp indir: %#v", rp.InDir)

	return rp
}
