package cache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"

	"github.com/Eric-Guo/sponge/pkg/encoding"
)

type options struct {
	numCounters int64
	maxCost     int64
	bufferItems int64
}

func defaultOptions() *options {
	return &options{
		numCounters: 1e7,     // number of keys to track frequency of (10M).
		maxCost:     1 << 30, // maximum cost of cache (1GB).
		bufferItems: 64,      // number of keys per Get buffer.
	}
}

// Option set the jwt options.
type Option func(*options)

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithNumCounters set number of keys.
func WithNumCounters(numCounters int64) Option {
	return func(o *options) {
		o.numCounters = numCounters
	}
}

// WithMaxCost set maximum cost of cache.
func WithMaxCost(maxCost int64) Option {
	return func(o *options) {
		o.maxCost = maxCost
	}
}

// WithBufferItems set number of keys per Get buffer.
func WithBufferItems(bufferItems int64) Option {
	return func(o *options) {
		o.bufferItems = bufferItems
	}
}

// InitMemory create a memory cache
func InitMemory(opts ...Option) *ristretto.Cache {
	o := defaultOptions()
	o.apply(opts...)

	// see: https://dgraph.io/blog/post/introducing-ristretto-high-perf-go-cache/
	//		https://www.start.io/blog/we-chose-ristretto-cache-for-go-heres-why/
	config := &ristretto.Config{
		NumCounters: o.numCounters,
		MaxCost:     o.maxCost,
		BufferItems: o.bufferItems,
	}
	store, err := ristretto.NewCache(config)
	if err != nil {
		panic(err)
	}
	return store
}

// ----------------------------------------------------------------------------

// global memory cache client
var (
	memoryCli *ristretto.Cache
	once      sync.Once
)

// InitGlobalMemory init global memory cache
func InitGlobalMemory(opts ...Option) {
	memoryCli = InitMemory(opts...)
}

// GetGlobalMemoryCli get memory cache client
func GetGlobalMemoryCli() *ristretto.Cache {
	if memoryCli == nil {
		once.Do(func() {
			memoryCli = InitMemory() // default options
		})
	}
	return memoryCli
}

// CloseGlobalMemory close memory cache
func CloseGlobalMemory() error {
	if memoryCli != nil {
		memoryCli.Close()
	}
	return nil
}

// ----------------------------------------------------------------------------

type memoryCache struct {
	client            *ristretto.Cache
	KeyPrefix         string
	encoding          encoding.Encoding
	DefaultExpireTime time.Duration
	newObject         func() interface{}
}

// NewMemoryCache create a memory cache
func NewMemoryCache(keyPrefix string, encode encoding.Encoding, newObject func() interface{}) Cache {
	return &memoryCache{
		client:    GetGlobalMemoryCli(),
		KeyPrefix: keyPrefix,
		encoding:  encode,
		newObject: newObject,
	}
}

// Set data
func (m *memoryCache) Set(_ context.Context, key string, val interface{}, expiration time.Duration) error {
	buf, err := encoding.Marshal(m.encoding, val)
	if err != nil {
		return fmt.Errorf("encoding.Marshal error: %v, key=%s, val=%+v ", err, key, val)
	}
	if len(buf) == 0 {
		buf = NotFoundPlaceholderBytes
	}
	cacheKey, err := BuildCacheKey(m.KeyPrefix, key)
	if err != nil {
		return fmt.Errorf("BuildCacheKey error: %v, key=%s", err, key)
	}
	ok := m.client.SetWithTTL(cacheKey, buf, 0, expiration)
	if !ok {
		return errors.New("SetWithTTL failed")
	}
	m.client.Wait()

	return nil
}

// Get data
func (m *memoryCache) Get(_ context.Context, key string, val interface{}) error {
	cacheKey, err := BuildCacheKey(m.KeyPrefix, key)
	if err != nil {
		return fmt.Errorf("BuildCacheKey error: %v, key=%s", err, key)
	}

	data, ok := m.client.Get(cacheKey)
	if !ok {
		return CacheNotFound // not found, convert to redis nil error
	}

	dataBytes, ok := data.([]byte)
	if !ok {
		return fmt.Errorf("data type error, key=%s, type=%T", key, data)
	}

	if len(dataBytes) == 0 || bytes.Equal(dataBytes, NotFoundPlaceholderBytes) {
		return ErrPlaceholder
	}

	err = encoding.Unmarshal(m.encoding, dataBytes, val)
	if err != nil {
		return fmt.Errorf("encoding.Unmarshal error: %v, key=%s, cacheKey=%s, type=%T, data=%s ",
			err, key, cacheKey, val, dataBytes)
	}
	return nil
}

// Del delete data
func (m *memoryCache) Del(_ context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	key := keys[0]
	cacheKey, err := BuildCacheKey(m.KeyPrefix, key)
	if err != nil {
		return fmt.Errorf("build cache key error, err=%v, key=%s", err, key)
	}
	m.client.Del(cacheKey)
	return nil
}

// MultiSet multiple set data
func (m *memoryCache) MultiSet(ctx context.Context, valueMap map[string]interface{}, expiration time.Duration) error {
	var err error
	for key, value := range valueMap {
		err = m.Set(ctx, key, value, expiration)
		if err != nil {
			return err
		}
	}
	return nil
}

// MultiGet multiple get data
func (m *memoryCache) MultiGet(ctx context.Context, keys []string, value interface{}) error {
	valueMap := reflect.ValueOf(value)
	var err error
	for _, key := range keys {
		object := m.newObject()
		err = m.Get(ctx, key, object)
		if err != nil {
			continue
		}
		valueMap.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(object))
	}

	return nil
}

// SetCacheWithNotFound set not found
func (m *memoryCache) SetCacheWithNotFound(_ context.Context, key string) error {
	cacheKey, err := BuildCacheKey(m.KeyPrefix, key)
	if err != nil {
		return fmt.Errorf("BuildCacheKey error: %v, key=%s", err, key)
	}

	ok := m.client.SetWithTTL(cacheKey, []byte(NotFoundPlaceholder), 0, DefaultNotFoundExpireTime)
	if !ok {
		return errors.New("SetWithTTL failed")
	}

	return nil
}
