package main

import (
	"fmt"
	"net/http"
	"os"
)

var kdb *KeyDB

func main() {

	args := os.Args

	if len(args) < 2 {
		fmt.Println("Usage: go run main.go <command>")
		os.Exit(1)
	}

	kdb = NewKeyDB("localhost:6379")

	switch args[1] {
	case "pp":
		fmt.Println("Starting the payment processor server...")
		paymentProcessor()
	case "ps":
		fmt.Println("Starting the payment summary server...")
		paymentSummary()
	}
}

func paymentProcessor() {
	mux := http.NewServeMux()
	mux.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}
		handlePaymentProcessor(w, r)
	})
	server := &http.Server{
		Addr:    ":9099",
		Handler: mux,
		HTTP2: &http.HTTP2Config{
			MaxConcurrentStreams: 100,
		},
	}

	go checkCurrentPayment()
	go channelSubscriber()

	server.ListenAndServe()
}

func paymentSummary() {
	mux := http.NewServeMux()
	mux.HandleFunc("/payments-summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}
		handlePaymentSummary(w, r)

	})

	server := &http.Server{
		Addr:    ":9099",
		Handler: mux,
		HTTP2: &http.HTTP2Config{
			MaxConcurrentStreams: 1000,
		},
	}
	if LOG_ON {
		fmt.Println("Server is running on port 9099")
	}

	// Inicia o listener de mensagens em background
	go listenAndSummarize()

	server.ListenAndServe()
}
