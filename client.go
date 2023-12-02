package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func getExchangeRate(ctx context.Context) (float64, error) {
	client := &http.Client{
		Timeout: 300 * time.Millisecond,
	}

	resp, err := client.Get("http://localhost:8080/cotacao")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result map[string]float64
	err = json.Unmarshal(body, &result)
	if err != nil {
		return 0, err
	}

	return result["bid"], nil
}

func saveToFile(value float64) error {
	content := fmt.Sprintf("Dólar: %f\n", value)
	return ioutil.WriteFile("cotacao.txt", []byte(content), 0644)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	exchangeRate, err := getExchangeRate(ctx)
	if err != nil {
		fmt.Println("Erro ao obter cotação:", err)
		return
	}

	err = saveToFile(exchangeRate)
	if err != nil {
		fmt.Println("Erro ao salvar no arquivo:", err)
		return
	}

	fmt.Println("Cotação do dólar salva com sucesso.")
}
