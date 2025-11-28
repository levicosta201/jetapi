package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ctx    context.Context
	ttl    time.Duration
}

type CacheConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	TTL      time.Duration // Time to live padrão em horas
}

func NewCache(config *CacheConfig) (*Cache, error) {
	if config == nil {
		return nil, fmt.Errorf("cache config cannot be nil")
	}

	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	if config.Host == "" {
		addr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.Password,
		DB:       config.DB,
	})

	ctx := context.Background()

	// Testar conexão
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour // Padrão: 24 horas
	}

	return &Cache{
		client: rdb,
		ctx:    ctx,
		ttl:    ttl,
	}, nil
}

// GenerateKey cria uma chave única baseada nos parâmetros da query
func (c *Cache) GenerateKey(prefix string, data interface{}) string {
	jsonData, _ := json.Marshal(data)
	hash := md5.Sum(jsonData)
	return fmt.Sprintf("%s:%x", prefix, hash)
}

// Get recupera um valor do cache
func (c *Cache) Get(key string, dest interface{}) (bool, error) {
	val, err := c.client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Chave não existe
	}
	if err != nil {
		return false, fmt.Errorf("error getting from cache: %v", err)
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return false, fmt.Errorf("error unmarshaling cached data: %v", err)
	}

	return true, nil
}

// Set armazena um valor no cache
func (c *Cache) Set(key string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	err = c.client.Set(c.ctx, key, jsonData, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("error setting cache: %v", err)
	}

	return nil
}

// SetWithTTL armazena um valor no cache com TTL customizado
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	err = c.client.Set(c.ctx, key, jsonData, ttl).Err()
	if err != nil {
		return fmt.Errorf("error setting cache: %v", err)
	}

	return nil
}

// Delete remove uma chave do cache
func (c *Cache) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

// Clear limpa todas as chaves com um prefixo específico
func (c *Cache) Clear(prefix string) error {
	iter := c.client.Scan(c.ctx, 0, prefix+":*", 0).Iterator()
	for iter.Next(c.ctx) {
		c.client.Del(c.ctx, iter.Val())
	}
	return iter.Err()
}

// Close fecha a conexão com Redis
func (c *Cache) Close() error {
	return c.client.Close()
}

// IsAvailable verifica se o cache está disponível
func (c *Cache) IsAvailable() bool {
	_, err := c.client.Ping(c.ctx).Result()
	return err == nil
}

