package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func fetchExchangeRate(ctx context.Context) (float64, error) {
	// Configurar cliente HTTP
	client := &http.Client{
		Timeout: 200 * time.Millisecond,
	}

	// Fazer a requisição à API
	resp, err := client.Get("https://economia.awesomeapi.com.br/json/last/USD-BRL")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Ler a resposta
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Analisar o JSON
	var result map[string]map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	return result["USDBRL"]["bid"], nil
}

func handleCotacao(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancel()

	dolar, err := fetchExchangeRate(ctx)
	if err != nil {
		http.Error(w, "Erro ao obter cotação do dólar", http.StatusInternalServerError)
		return
	}

	// Inserir no banco de dados
	insertCtx, insertCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer insertCancel()
	_, err = db.ExecContext(insertCtx, "INSERT INTO cotacoes (dolar) VALUES ($1)", dolar)
	if err != nil {
		http.Error(w, "Erro ao inserir no banco de dados", http.StatusInternalServerError)
		return
	}

	// Retornar a cotação para o cliente
	response := map[string]float64{"bid": dolar}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Erro ao gerar resposta JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func main() {
	http.HandleFunc("/cotacao", handleCotacao)
	http.ListenAndServe(":8080", nil)
}
