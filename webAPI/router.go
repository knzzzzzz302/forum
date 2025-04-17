package webAPI

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	t, err := template.ParseFiles("public/HTML/404.html")
	if err != nil {
		http.Error(w, "Page non trouvée", http.StatusNotFound)
		return
	}
	t.Execute(w, nil)
}

var Debug = true

func DebugPrintf(format string, a ...interface{}) {
	if Debug {
		fmt.Printf("[DEBUG] "+format+"\n", a...)
	}
}


type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int           
	window   time.Duration 
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	DebugPrintf("Création d'un nouveau RateLimiter : %d requêtes par %v", limit, window)
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	
	requests := rl.requests[ip]
	var cleaned []time.Time

	for _, req := range requests {
		if now.Sub(req) <= rl.window {
			cleaned = append(cleaned, req)
		}
	}

	if len(cleaned) >= rl.limit {
		DebugPrintf("🚫 RATE LIMIT: IP %s BLOQUÉE - %d requêtes dans la fenêtre", ip, len(cleaned))
		return false
	}

	
	rl.requests[ip] = append(cleaned, now)
	DebugPrintf("✅ RATE LIMIT: IP %s autorisée - %d requêtes actuelles", ip, len(rl.requests[ip]))
	return true
}

type CustomRouter struct {
	routes      map[string]http.HandlerFunc
	static      http.Handler
	rateLimiter *RateLimiter
}

func NewCustomRouter() *CustomRouter {
	DebugPrintf("Création d'un nouveau CustomRouter avec RateLimiter")
	return &CustomRouter{
		routes:      make(map[string]http.HandlerFunc),
		static:      http.FileServer(http.Dir("public")),
		rateLimiter: NewRateLimiter(150, 1*time.Minute), 
	}
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		DebugPrintf("IP via X-Forwarded-For: %s", ip)
		return ip
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		DebugPrintf("IP via X-Real-IP: %s", ip)
		return ip
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		DebugPrintf("Erreur lors de la récupération de l'IP : %v", err)
		ip = r.RemoteAddr
	}

	DebugPrintf("IP via RemoteAddr: %s", ip)
	return ip
}

func (r *CustomRouter) HandleFunc(path string, handler http.HandlerFunc) {
	DebugPrintf("Ajout d'une route : %s", path)
	r.routes[path] = func(w http.ResponseWriter, req *http.Request) {
		
		DebugPrintf("Requête reçue : Path=%s, Method=%s", path, req.Method)

		ip := getClientIP(req)
		
		if !r.rateLimiter.Allow(ip) {
			DebugPrintf("🚨 BLOQUÉ: Requête de %s rejetée", ip)
			http.Error(w, "Trop de requêtes. Veuillez réessayer plus tard.", http.StatusTooManyRequests)
			return
		}
		
		handler(w, req)
	}
}

func (r *CustomRouter) Handle(path string, handler http.Handler) {
	DebugPrintf("Ajout d'un gestionnaire : %s", path)
	r.routes[path] = func(w http.ResponseWriter, req *http.Request) {
		
		DebugPrintf("Requête reçue : Path=%s, Method=%s", path, req.Method)

		ip := getClientIP(req)
		
		if !r.rateLimiter.Allow(ip) {
			DebugPrintf("🚨 BLOQUÉ: Requête de %s rejetée", ip)
			http.Error(w, "Trop de requêtes. Veuillez réessayer plus tard.", http.StatusTooManyRequests)
			return
		}
		
		handler.ServeHTTP(w, req)
	}
}

func (r *CustomRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	DebugPrintf("ServeHTTP appelé pour : %s", req.URL.Path)

	if strings.HasPrefix(req.URL.Path, "/public/") {
		DebugPrintf("Route statique détectée : %s", req.URL.Path)
		handler, ok := r.routes["/public/"]
		if ok {
			handler(w, req)
			return
		}
	}

	handler, ok := r.routes[req.URL.Path]
	if ok {
		DebugPrintf("Route trouvée : %s", req.URL.Path)
		handler(w, req)
		return
	}

	DebugPrintf("Route non trouvée : %s", req.URL.Path)
	NotFoundHandler(w, req)
}