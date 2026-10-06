package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/Misaka8339-ctrl/chirpy/internal/database"
)

// 保存服务器运行期间的共享状态
type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	tokenSecret    string
	polkaKey       string
}

func main() {
	// 加载本地环境变量
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	polkaKey := os.Getenv("POLKA_KEY")
	if polkaKey == "" {
		log.Fatal("POLKA_KEY is not set")
	}

	tokenSecret := os.Getenv("JWT_SECRET")
	if tokenSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	// 创建数据库连接池
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}
	defer db.Close()

	// 实际验证数据库是否可连接
	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	apiCfg := &apiConfig{
		db:          database.New(db),
		platform:    os.Getenv("PLATFORM"),
		tokenSecret: tokenSecret,
		polkaKey:    polkaKey,
	}

	mux := http.NewServeMux()

	// 给文件服务套上计数中间件
	fileHandler := http.StripPrefix(
		"/app",
		http.FileServer(http.Dir(".")),
	)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileHandler))

	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)

	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)

	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)

	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirp)

	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerPolkaWebhook)

	// 配置 HTTP 服务器
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server listening on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Server stopped: %v", err)
	}
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

// 中间件：先增加计数，再交给原来的处理器
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// 查看计数
func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, cfg.fileserverHits.Load())

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// 重置计数
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	// 必须先检查环境，再执行任何重置操作
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Forbidden")
		return
	}

	if err := cfg.db.DeleteAllUsers(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset users")
		return
	}

	// 保留原来的访问次数清零功能
	cfg.fileserverHits.Store(0)

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Users deleted and metrics reset",
	})
}
