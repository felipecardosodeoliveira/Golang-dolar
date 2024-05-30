package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	serverURL      = "http://localhost:8080/cotacao"
	clientTimeout  = 300 * time.Millisecond
	outputFilename = "cotacao.txt"
)

func fetchCotacaoByServer(ctx context.Context) (string, error) {
	client := http.Client{Timeout: clientTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição HTTP: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao realizar requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	var bid string
	if err := json.NewDecoder(resp.Body).Decode(&bid); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta JSON: %v", err)
	}

	return bid, nil
}

func saveCotacaoToFile(bid string) error {
	content := fmt.Sprintf("Dólar: %s", bid)
	if err := os.WriteFile(outputFilename, []byte(content), 0644); err != nil {
		return fmt.Errorf("erro ao salvar no arquivo: %v", err)
	}
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()

	cotacao, err := fetchCotacaoByServer(ctx)
	if err != nil {
		fmt.Println("Erro ao obter cotação do servidor:", err)
		return
	}

	if err := saveCotacaoToFile(cotacao); err != nil {
		fmt.Println("Erro ao salvar no arquivo:", err)
		return
	}

	fmt.Printf("Cotação atual salva em %s\n", outputFilename)
}
