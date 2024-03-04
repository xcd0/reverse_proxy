package main

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Basic認証をチェックする関数
func BasicAuth(next http.Handler, authConfig *Auth) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, authConfig.Path) {
			user, pass, ok := r.BasicAuth()

			if ok {
				if user == authConfig.UserName && bcrypt.CompareHashAndPassword([]byte(authConfig.Password), []byte(pass)) == nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
