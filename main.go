package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
)

var kdb *KeyDB

func main() {
	// Configurações do Garbage Collector
	setupGarbageCollector()

	args := os.Args

	if len(args) < 2 {
		fmt.Println("Usage: go run main.go <command>")
		os.Exit(1)
	}

	kdb = NewKeyDB("localhost:6379")
	defer kdb.Close()

	// Setup graceful shutdown
	setupGracefulShutdown()

	switch args[1] {
	case "pp":
		fmt.Println("Starting the payment processor server...")
		paymentProcessor()
	case "ps":
		fmt.Println("Starting the payment summary server...")
		paymentSummary()
	}
}

// Configurações do Garbage Collector
func setupGarbageCollector() {
	// Força GC a cada 100MB alocados
	debug.SetGCPercent(100)

	// Configura o número máximo de CPUs
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Configura o tamanho da stack das goroutines
	debug.SetMaxStack(32 * 1024 * 1024) // 32MB

	fmt.Printf("GC configurado: GCPercent=%d, GOMAXPROCS=%d\n",
		debug.SetGCPercent(-1), runtime.GOMAXPROCS(0))
}

// Setup graceful shutdown
func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nRecebido sinal de shutdown, finalizando...")
		shutdownGracefully()
		os.Exit(0)
	}()
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

	fmt.Println("Payment processor server running on port 9099")
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

	fmt.Println("Payment summary server running on port 9099")
	server.ListenAndServe()
}
