package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/microshop/pkg/config"
	"github.com/go-redis/redis/v8"
)

type RedisRepository struct {
	client *redis.Client
	config *config.RedisConfig
}

func NewRedisRepository(cfg *config.RedisConfig) *RedisRepository {
	return &RedisRepository{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
			PoolSize: cfg.PoolSize,
		}),
		config: cfg,
	}
}

func (r *RedisRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisRepository) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisRepository) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisRepository) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisRepository) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisRepository) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (r *RedisRepository) Close() error {
	return r.client.Close()
}

// Cache for user data
type UserCache struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (r *RedisRepository) CacheUser(ctx context.Context, user *UserCache) error {
	key := fmt.Sprintf("user:%s", user.ID)
	return r.SetJSON(ctx, key, user, 30*time.Minute)
}

func (r *RedisRepository) GetUserCache(ctx context.Context, userID string) (*UserCache, error) {
	key := fmt.Sprintf("user:%s", userID)
	var user UserCache
	err := r.GetJSON(ctx, key, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
