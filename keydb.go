package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type KeyDB struct {
	conn *redis.Client
}

func NewKeyDB(addr string) *KeyDB {
	conn := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		Password:     "", // no password set
		DB:           0,  // use default DB
		PoolSize:     10, // Pool de conexões
		MinIdleConns: 5,  // Mínimo de conexões idle
		MaxRetries:   3,  // Máximo de tentativas
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	return &KeyDB{
		conn: conn,
	}
}

func (k *KeyDB) Get(ctx context.Context, key string) (string, error) {
	return k.conn.Get(ctx, key).Result()
}

func (k *KeyDB) Set(ctx context.Context, key string, value string) error {
	return k.conn.Set(ctx, key, value, 0).Err()
}

func (k *KeyDB) Publish(ctx context.Context, message string) error {
	return k.conn.Publish(ctx, "payments", message).Err()
}

func (k *KeyDB) Subscribe(ctx context.Context) <-chan *redis.Message {
	pubsub := k.conn.Subscribe(ctx, "payments")
	return pubsub.Channel()
}

// Função para fechar conexões adequadamente
func (k *KeyDB) Close() error {
	return k.conn.Close()
}
