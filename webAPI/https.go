package webAPI

import (
	"net/http"
	"strings"
)

func RedirectToHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			host := r.Host
			if strings.Contains(host, ":") {
				host = strings.Split(host, ":")[0] + ":443"
			} 
			http.Redirect(w, r, "https://"+host+r.RequestURI, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}