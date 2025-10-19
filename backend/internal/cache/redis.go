package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrCacheMiss = errors.New("cache miss")
	ErrCacheDown = errors.New("cache unavailable")
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

type CacheConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

func NewRedisCache(config *CacheConfig) *RedisCache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	return &RedisCache{
		client: rdb,
		ctx:    context.Background(),
	}
}

func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	ctx, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()

	err = r.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

func (r *RedisCache) Get(key string, dest interface{}) error {
	ctx, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	return nil
}

func (r *RedisCache) Delete(key string) error {
	ctx, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()

	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) DeletePattern(pattern string) error {
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}

	return nil
}

func (r *RedisCache) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 3*time.Second)
	defer cancel()

	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return result > 0, nil
}

func (r *RedisCache) SetWithTags(key string, value interface{}, expiration time.Duration, tags []string) error {
	if err := r.Set(key, value, expiration); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	pipe := r.client.Pipeline()
	for _, tag := range tags {
		tagKey := fmt.Sprintf("tag:%s", tag)
		pipe.SAdd(ctx, tagKey, key)
		pipe.Expire(ctx, tagKey, expiration)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCache) InvalidateByTag(tag string) error {
	ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)
	defer cancel()

	tagKey := fmt.Sprintf("tag:%s", tag)

	keys, err := r.client.SMembers(ctx, tagKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get tag members: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	allKeys := append(keys, tagKey)
	return r.client.Del(ctx, allKeys...).Err()
}

func (r *RedisCache) Health() error {
	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) Stats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	info, err := r.client.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	poolStats := r.client.PoolStats()

	return map[string]interface{}{
		"redis_info":    info,
		"pool_hits":     poolStats.Hits,
		"pool_misses":   poolStats.Misses,
		"pool_timeouts": poolStats.Timeouts,
		"pool_total":    poolStats.TotalConns,
		"pool_idle":     poolStats.IdleConns,
		"pool_stale":    poolStats.StaleConns,
	}
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
