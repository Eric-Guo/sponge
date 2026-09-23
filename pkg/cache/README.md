## cache

redis and memory cache libraries.

### Example of use

#### Using Redis Cache

```go
package main

import (
	"github.com/redis/go-redis/v9"
	"github.com/Eric-Guo/sponge/pkg/cache"
	"github.com/Eric-Guo/sponge/pkg/encoding"
)

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	cachePrefix := ""
	jsonEncoding := encoding.JSONEncoding{}
	newObject := func() interface{} {
		return &User{}
	}

	c := cache.NewRedisCache(redisClient, cachePrefix, jsonEncoding, newObject)
	// operations
	// c.Set(ctx, key, value, expiration)
	// c.Get(ctx, key)
	// c.Delete(ctx, key)
}
```

#### Using Memory Cache

```go
package main

import (
	"github.com/Eric-Guo/sponge/pkg/cache"
	"github.com/Eric-Guo/sponge/pkg/encoding"
)

func main() {
	// set memory cache client
	//cache.InitGlobalMemory(
	//	WithNumCounters(1e7),
	//	WithMaxCost(1<<30),
	//	WithBufferItems(64),
	//)

	cachePrefix := ""
	jsonEncoding := encoding.JSONEncoding{}
	newObject := func() interface{} {
		return &User{}
	}

	c := cache.NewMemoryCache(redisClient, cachePrefix, jsonEncoding, newObject)
	// operations
	// c.Set(ctx, key, value, expiration)
	// c.Get(ctx, key)
	// c.Delete(ctx, key)
}
```
