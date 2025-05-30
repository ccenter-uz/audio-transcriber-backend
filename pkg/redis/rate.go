package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func IsRequestAllowed(ctx context.Context, rdb *redis.Client, userID string, limit int, windowSeconds int, blockDurationSeconds int) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s", userID)
	blockKey := fmt.Sprintf("block:%s", userID)

	blocked, err := rdb.Exists(ctx, blockKey).Result()
	if err != nil {
		return false, err
	}
	if blocked == 1 {
		return false, nil 
	}

	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		rdb.Expire(ctx, key, time.Duration(windowSeconds)*time.Second)
	}

	if count > int64(limit) {
		rdb.Set(ctx, blockKey, "1", time.Duration(blockDurationSeconds)*time.Second)
		return false, nil
	}

	return true, nil
}
