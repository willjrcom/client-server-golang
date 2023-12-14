package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cotacoes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dolar REAL,
			registro TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		panic(err)
	}
}

func fetchExchangeRate(ctx context.Context) (any, error) {
	// Configurar cliente HTTP
	client := &http.Client{
		Timeout: 2 * time.Second, // Aumentado o tempo limite para 2 segundos
	}

	// Fazer a requisição à API
	resp, err := client.Get("https://economia.awesomeapi.com.br/json/last/USD-BRL")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Ler a resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Analisar o JSON
	var result map[string]map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	return result["USDBRL"]["bid"], nil
}

func handleCotacao(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // Aumentado o tempo limite para 5 segundos
	defer cancel()

	dolar, err := fetchExchangeRate(ctx)
	if err != nil {
		http.Error(w, "Erro ao obter cotação do dólar: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Inserir no banco de dados
	insertCtx, insertCancel := context.WithTimeout(context.Background(), 1*time.Second) // Aumentado o tempo limite para 1 segundo
	defer insertCancel()
	_, err = db.ExecContext(insertCtx, "INSERT INTO cotacoes (dolar) VALUES ($1)", dolar)
	if err != nil {
		http.Error(w, "Erro ao inserir no banco de dados: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar a cotação para o cliente
	response := map[string]any{"bid": dolar}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Erro ao gerar resposta JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func main() {
	http.HandleFunc("/cotacao", handleCotacao)

	// Verificar se há erros ao iniciar o servidor
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
