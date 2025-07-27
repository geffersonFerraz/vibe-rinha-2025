package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Request struct {
	CorrelationID string  `json:"correlation_id"`
	Amount        float64 `json:"amount"`
}

type RequestToPaymentProcessor struct {
	Request
	RequestedAt time.Time `json:"requestedAt"`
}

var LOG_ON = os.Getenv("LOG_ON") == "true"
var TIME_TO_CHECK_PAYMENT_PROCESSOR = 4998 * time.Millisecond

var CURRENT_PAYMENT_PROCESSOR = 0
var ENABLE_QUEUE_PROCESSOR = false

type Processor struct {
	URL             string
	Failing         bool
	MinResponseTime int
	LastCheck       time.Time
}

var PROCESSOR_LIST = []Processor{
	{
		URL:             "QUEUE_PROCESSOR",
		Failing:         false,
		MinResponseTime: 0,
		LastCheck:       time.Now().Add(-1 * time.Hour),
	},
	{
		URL:             os.Getenv("PAYMENT_PROCESSOR_URL"),
		Failing:         false,
		MinResponseTime: 0,
		LastCheck:       time.Now().Add(-1 * time.Hour),
	},
	{
		URL:             os.Getenv("PAYMENT_PROCESSOR_FALLBACK_URL"),
		Failing:         false,
		MinResponseTime: 0,
		LastCheck:       time.Now().Add(-1 * time.Hour),
	},
}

type ServiceHealthResponse struct {
	Failing         bool `json:"failing"`
	MinResponseTime int  `json:"minResponseTime"`
}

func checkCurrentPayment() {
	lastCheck := time.Now().Add(-1 * time.Hour)
	timeout := 2000 * time.Millisecond

	for {
		if time.Since(lastCheck) < TIME_TO_CHECK_PAYMENT_PROCESSOR {
			time.Sleep(TIME_TO_CHECK_PAYMENT_PROCESSOR - time.Since(lastCheck))
		}

		// Canais para receber resultados com timeout
		defaultChan := make(chan ServiceHealthResponse, 1)
		fallbackChan := make(chan ServiceHealthResponse, 1)

		// Inicia as verificações em paralelo
		go func() {
			defaultChan <- checkCurrentURLWithTimeout(PROCESSOR_LIST[1].URL, timeout)
		}()
		go func() {
			fallbackChan <- checkCurrentURLWithTimeout(PROCESSOR_LIST[2].URL, timeout)
		}()

		// Aguarda ambos os resultados com timeout
		defaultProcessor := <-defaultChan
		fallbackProcessor := <-fallbackChan

		lastCheck = time.Now()

		// Caso ambos os processadores estejam falhando, usa a fila
		if defaultProcessor.Failing && fallbackProcessor.Failing {
			CURRENT_PAYMENT_PROCESSOR = 0
			ENABLE_QUEUE_PROCESSOR = true
			continue
		}

		// Caso o processador principal esteja funcionando e tenha um tempo de resposta menor que 3 segundos, usa ele
		if !defaultProcessor.Failing && defaultProcessor.MinResponseTime <= 3000 {
			CURRENT_PAYMENT_PROCESSOR = 1
			ENABLE_QUEUE_PROCESSOR = false
			continue
		}

		// Caso o processador fallback esteja funcionando e tenha um tempo de resposta menor que 3 segundos, usa ele
		if defaultProcessor.Failing && fallbackProcessor.MinResponseTime <= 3000 {
			CURRENT_PAYMENT_PROCESSOR = 2
			ENABLE_QUEUE_PROCESSOR = false
			continue
		}

		// Caso nenhum dos processadores esteja funcionando adequadamente, usa a fila
		CURRENT_PAYMENT_PROCESSOR = 0
		ENABLE_QUEUE_PROCESSOR = true
	}
}

func checkCurrentURLWithTimeout(url string, timeout time.Duration) ServiceHealthResponse {
	// Canal para receber o resultado
	resultChan := make(chan ServiceHealthResponse, 1)

	// Executa a verificação em uma goroutine
	go func() {
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get(url + "/payments/service-health")
		if err != nil {
			resultChan <- ServiceHealthResponse{
				Failing:         true,
				MinResponseTime: 0,
			}
			return
		}
		defer resp.Body.Close()

		var serviceHealthResponse ServiceHealthResponse
		err = json.NewDecoder(resp.Body).Decode(&serviceHealthResponse)
		if err != nil {
			resultChan <- ServiceHealthResponse{
				Failing:         true,
				MinResponseTime: 0,
			}
			return
		}
		resultChan <- serviceHealthResponse
	}()

	// Aguarda o resultado ou timeout
	select {
	case result := <-resultChan:
		return result
	case <-time.After(timeout):
		return ServiceHealthResponse{
			Failing:         true,
			MinResponseTime: 9999,
		}
	}
}

var ch = make(chan Request)

func channelPublisher(amount float64, correlationID string) {
	go func() {
		ch <- Request{
			Amount:        amount,
			CorrelationID: correlationID,
		}
	}()
}

func serializeToBytes(msg RequestToPaymentProcessor) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(msg)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func deserializeFromBytes(data []byte) (RequestToPaymentProcessor, error) {
	var msg RequestToPaymentProcessor
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&msg)
	return msg, err
}

func channelSubscriber() {
	go func() {
		for {
			msg := <-ch
			if ENABLE_QUEUE_PROCESSOR {
				channelPublisher(msg.Amount, msg.CorrelationID)
				continue
			}

			request := RequestToPaymentProcessor{
				Request:     msg,
				RequestedAt: time.Now(),
			}
			bytesToPublish, err := serializeToBytes(request)
			if err != nil {
				fmt.Println(err)
			}

			// make a request to the payment processor
			jsonReq, err := json.Marshal(request)
			if err != nil {
				fmt.Println(err)
			}
			resp, err := http.Post(
				PROCESSOR_LIST[CURRENT_PAYMENT_PROCESSOR].URL+"/payments",
				"application/json",
				bytes.NewBuffer(jsonReq),
			)
			if err != nil {
				fmt.Println(err)
			}
			resp.Body.Close()

			kdb.Publish(context.Background(), string(bytesToPublish))
		}
	}()
}

func handlePaymentProcessor(w http.ResponseWriter, r *http.Request) {
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	channelPublisher(req.Amount, req.CorrelationID)

	w.WriteHeader(http.StatusNoContent)
}
