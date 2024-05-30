package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	apiURL        = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	dbPath        = "data.db"
	apiTimeout    = 200 * time.Millisecond
	dbTimeout     = 10 * time.Millisecond
	clientTimeout = 300 * time.Millisecond
	serverPort    = ":8080"
)

type Cotacao struct {
	USDBRL struct {
		Ask        string `json:"ask"`
		Bid        string `json:"bid"`
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		CreateDate string `json:"create_date"`
		High       string `json:"high"`
		Low        string `json:"low"`
		Name       string `json:"name"`
		PctChange  string `json:"pctChange"`
		Timestamp  string `json:"timestamp"`
		VarBid     string `json:"varBid"`
	} `json:"USDBRL"`
}

func getDolarPrice(ctx context.Context) (Cotacao, error) {
	client := http.Client{Timeout: apiTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Cotacao{}, fmt.Errorf("erro ao criar requisição HTTP: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Cotacao{}, fmt.Errorf("erro ao realizar requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	var cotacao Cotacao
	if err := json.NewDecoder(resp.Body).Decode(&cotacao); err != nil {
		return Cotacao{}, fmt.Errorf("erro ao decodificar resposta JSON: %v", err)
	}

	return cotacao, nil
}

func saveDatabase(ctx context.Context, db *sql.DB, cotacao Cotacao) error {
	_, err := db.ExecContext(ctx, "INSERT INTO cotacoes (code, codein, name, high, low, varBid, pctChange, bid, ask, timestamp, create_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		cotacao.USDBRL.Code, cotacao.USDBRL.Codein, cotacao.USDBRL.Name, cotacao.USDBRL.High, cotacao.USDBRL.Low,
		cotacao.USDBRL.VarBid, cotacao.USDBRL.PctChange, cotacao.USDBRL.Bid, cotacao.USDBRL.Ask,
		cotacao.USDBRL.Timestamp, cotacao.USDBRL.CreateDate)
	if err != nil {
		return fmt.Errorf("erro ao salvar cotação no banco de dados: %v", err)
	}
	return nil
}

func handleCotacao(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx, cancel := context.WithTimeout(r.Context(), clientTimeout)
	defer cancel()

	cotacao, err := getDolarPrice(ctx)
	if err != nil {
		log.Printf("erro ao obter cotação: %v\n", err)
		http.Error(w, "Erro ao obter cotação", http.StatusInternalServerError)
		return
	}

	ctx, cancel = context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	if err := saveDatabase(ctx, db, cotacao); err != nil {
		log.Printf("erro ao salvar cotação no banco de dados: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cotacao.USDBRL.Bid)
}

func setupDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco de dados: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cotacoes 
	(id INTEGER PRIMARY KEY, 
	code TEXT, 
	codein TEXT, 
	name TEXT,
	high TEXT, 
	low TEXT, 
	varBid TEXT, 
	pctChange TEXT, 
	bid TEXT, 
	ask TEXT, 
	timestamp TEXT, 
	create_date TEXT)`)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar tabela no banco de dados: %v", err)
	}

	return db, nil
}

func main() {
	db, err := setupDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		handleCotacao(w, r, db)
	})

	log.Printf("Servidor rodando na porta %s...\n", serverPort)
	log.Fatal(http.ListenAndServe(serverPort, nil))
}
