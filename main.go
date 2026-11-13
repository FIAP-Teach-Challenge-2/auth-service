package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

// App struct (para injeção de dependência)
type App struct {
	DB        *sql.DB
	MasterKey string
	Fault     *FaultSwitch
}

func main() {
	// Carrega o .env para desenvolvimento local. Em produção, isso não fará nada.
	_ = godotenv.Load()

	// Contexto encerrado no SIGTERM do Kubernetes (graceful shutdown + flush da telemetria)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- OpenTelemetry ---
	otelShutdown, err := setupOTel(ctx)
	if err != nil {
		log.Fatalf("Não foi possível inicializar o OpenTelemetry: %v", err)
	}

	// --- Configuração ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001" // Porta padrão
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL deve ser definida")
	}

	masterKey := os.Getenv("MASTER_KEY")
	if masterKey == "" {
		log.Fatal("MASTER_KEY deve ser definida")
	}

	// --- Conexão com o Banco ---
	db, err := connectDB(databaseURL)
	if err != nil {
		log.Fatalf("Não foi possível conectar ao banco de dados: %v", err)
	}
	defer func() { _ = db.Close() }()

	app := &App{
		DB:        db,
		MasterKey: masterKey,
		Fault:     &FaultSwitch{},
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Serviço de Autenticação (Go) rodando na porta %s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("Encerrando: aguardando requisições em andamento e enviando telemetria pendente...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Erro no shutdown do servidor HTTP: %v", err)
	}
	if err := otelShutdown(shutdownCtx); err != nil {
		log.Printf("Erro no shutdown do OpenTelemetry: %v", err)
	}
}

// routes monta as rotas da API envolvidas pelo otelhttp: span SERVER por
// requisição + métricas http.server.* (exceto /health, chamado pelos probes do
// Kubernetes). Respostas 5xx marcam o span como erro.
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.healthHandler)

	// Endpoint público para validar uma chave
	mux.HandleFunc("/validate", a.validateKeyHandler)

	// Endpoints de "admin" para criar/gerenciar chaves
	// Eles são protegidos pelo middleware de autenticação
	mux.Handle("/admin/keys", a.masterKeyAuthMiddleware(http.HandlerFunc(a.createKeyHandler)))

	if os.Getenv("CHAOS_ENABLED") == "true" {
		mux.Handle("/admin/chaos", a.masterKeyAuthMiddleware(http.HandlerFunc(a.chaosHandler)))
		log.Println("Endpoint de chaos habilitado em /admin/chaos")
	}

	return otelhttp.NewHandler(mux, serviceName,
		otelhttp.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/health" }),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}

// connectDB inicializa e testa a conexão com o PostgreSQL.
// otelsql envolve o driver pgx: cada query feita com *Context vira um span
// CLIENT (db.system=postgresql) filho da requisição HTTP.
func connectDB(databaseURL string) (*sql.DB, error) {
	db, err := otelsql.Open("pgx", databaseURL,
		otelsql.WithAttributes(attribute.String("db.system", "postgresql")),
	)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Conectado ao PostgreSQL com sucesso!")
	return db, nil
}
