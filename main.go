package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"syscall"
)

func setupGarbageCollector() {
	runtime.GOMAXPROCS(1)
	debug.SetMaxStack(32 * 1024 * 1024)
}

type Config struct {
	Listen  string `json:"listen"`
	Timeout int    `json:"timeout"`
	Debug   bool   `json:"debug"`
}

func loadConfig() (*Config, error) {
	var config Config
	config.Listen = "pp"
	if listen := os.Getenv("PP_LISTEN"); listen != "" {
		config.Listen = listen
	}

	config.Timeout = 1000
	if timeout := os.Getenv("PP_TIMEOUT"); timeout != "" {
		timeoutInt, err := strconv.Atoi(timeout)
		if err != nil {
			return nil, err
		}
		config.Timeout = timeoutInt
	}

	config.Debug = false
	if debug := os.Getenv("PP_DEBUG"); debug != "" {
		config.Debug = debug == "true"
	}

	log.Println("Config loaded: " + config.Listen)
	return &config, nil
}

func createSocketConnection(config *Config) (net.Listener, error) {
	log.Println("Creating socket connection...")
	socketFile := fmt.Sprintf("/tmp/rinha/socket-%s.sock", config.Listen)

	// Criar o diretório se não existir
	socketDir := "/tmp/rinha"
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remover socket anterior se existir
	syscall.Unlink(socketFile)

	conn, err := net.Listen("unix", socketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket listener: %w", err)
	}

	log.Printf("Socket created at: %s", socketFile)
	return conn, nil
}

func main() {
	setupGarbageCollector()

	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Config loaded: " + config.Listen)

	socketConn, err := createSocketConnection(config)
	if err != nil {
		log.Fatal(err)
	}
	defer socketConn.Close()

	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received message:")
		w.Write([]byte("{\"message\": \"Hello kung fu developer!\"}"))
	})

	m.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received message:")
		w.Write([]byte("{\"message\": \"Hello payments!\"}"))
	})

	m.HandleFunc("/payments-summary", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received message:")
		w.Write([]byte("{\"default\":{\"totalRequests\":43236,\"totalAmount\":415542345.98},\"fallback\":{\"totalRequests\":423545,\"totalAmount\":329347.34}}"))
	})

	server := http.Server{
		Handler: m,
	}

	if err := server.Serve(socketConn); err != nil {
		log.Fatal(err)
	}

}
