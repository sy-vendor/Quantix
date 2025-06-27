package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache Redis缓存客户端
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisCache 创建Redis缓存客户端
func NewRedisCache(host string, port int, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	return &RedisCache{
		client: client,
		ctx:    ctx,
	}, nil
}

// Set 设置缓存
func (c *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	return c.client.Set(c.ctx, key, data, expiration).Err()
}

// Get 获取缓存
func (c *RedisCache) Get(key string, dest interface{}) error {
	data, err := c.client.Get(c.ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("键不存在: %s", key)
		}
		return fmt.Errorf("获取缓存失败: %w", err)
	}

	return json.Unmarshal(data, dest)
}

// Delete 删除缓存
func (c *RedisCache) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

// Exists 检查键是否存在
func (c *RedisCache) Exists(key string) (bool, error) {
	result, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Close 关闭连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// GetOrSet 获取缓存，如果不存在则设置
func (c *RedisCache) GetOrSet(key string, dest interface{}, setter func() (interface{}, error), expiration time.Duration) error {
	// 尝试获取缓存
	err := c.Get(key, dest)
	if err == nil {
		return nil
	}

	// 缓存不存在，调用setter获取数据
	data, err := setter()
	if err != nil {
		return err
	}

	// 设置缓存
	if err := c.Set(key, data, expiration); err != nil {
		return err
	}

	// 将数据复制到dest
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(dataBytes, dest)
}
