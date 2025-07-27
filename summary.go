package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Summary struct {
	Default struct {
		TotalRequests int     `json:"totalRequests"`
		TotalAmount   float64 `json:"totalAmount"`
	} `json:"default"`
	Fallback struct {
		TotalRequests int     `json:"totalRequests"`
		TotalAmount   float64 `json:"totalAmount"`
	} `json:"fallback"`
}

var summary orderedmap.OrderedMap[string, RequestToPaymentProcessor]
var summaryMutex sync.RWMutex

// Configurações para limpeza de memória
const (
	CLEANUP_INTERVAL = 5 * time.Minute // Limpeza a cada 5 minutos
	MAX_SUMMARY_SIZE = 10000           // Máximo de 10k entradas
	MAX_AGE_HOURS    = 24              // Manter apenas dados das últimas 24h
)

func listenAndSummarize() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan := kdb.Subscribe(ctx)

	// Goroutine para limpeza periódica
	go cleanupSummaryPeriodically(ctx)

	for msg := range msgChan {
		request, err := deserializeFromBytes([]byte(msg.Payload))
		if err != nil {
			fmt.Printf("Erro ao deserializar mensagem: %v\n", err)
			continue
		}

		// Thread-safe append
		summaryMutex.Lock()
		summary.Set(request.CorrelationID, request)

		// Limpeza automática se exceder o tamanho máximo
		if summary.Len() > MAX_SUMMARY_SIZE {
			cleanupOldEntries()
		}
		summaryMutex.Unlock()

		if LOG_ON {
			fmt.Printf("Mensagem recebida: %+v\n", request)
		}
	}
}

// Limpeza periódica do mapa de resumo
func cleanupSummaryPeriodically(ctx context.Context) {
	ticker := time.NewTicker(CLEANUP_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			summaryMutex.Lock()
			cleanupOldEntries()
			summaryMutex.Unlock()

			if LOG_ON {
				fmt.Printf("Limpeza periódica executada. Tamanho atual: %d\n", summary.Len())
			}
		}
	}
}

// Remove entradas antigas (mais de 24 horas)
func cleanupOldEntries() {
	cutoffTime := time.Now().Add(-MAX_AGE_HOURS * time.Hour)

	// Remove entradas antigas
	for pair := summary.Oldest(); pair != nil; {
		next := pair.Next()
		if pair.Value.RequestedAt.Before(cutoffTime) {
			summary.Delete(pair.Key)
		}
		pair = next
	}

	// Se ainda estiver muito grande, remove as mais antigas
	for summary.Len() > MAX_SUMMARY_SIZE {
		if oldest := summary.Oldest(); oldest != nil {
			summary.Delete(oldest.Key)
		} else {
			break
		}
	}
}

func summarize(from, to time.Time) Summary {
	result := Summary{}

	summaryMutex.RLock()
	defer summaryMutex.RUnlock()

	for pair := summary.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value.RequestedAt.After(from) && pair.Value.RequestedAt.Before(to) {
			if pair.Value.Amount > 100 {
				result.Fallback.TotalRequests++
				result.Fallback.TotalAmount += pair.Value.Amount
			} else {
				result.Default.TotalRequests++
				result.Default.TotalAmount += pair.Value.Amount
			}
		}
	}
	return result
}

// GET /payments-summary?from=2020-07-10T12:34:56.000Z&to=2020-07-10T12:35:56.000Z
func handlePaymentSummary(w http.ResponseWriter, r *http.Request) {
	if LOG_ON {
		fmt.Println("handlePaymentSummary inside")
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Summary{})
		return
	}

	// Parse das datas
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		http.Error(w, "Data 'from' inválida", http.StatusBadRequest)
		return
	}

	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		http.Error(w, "Data 'to' inválida", http.StatusBadRequest)
		return
	}

	var result Summary = summarize(fromTime, toTime)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
