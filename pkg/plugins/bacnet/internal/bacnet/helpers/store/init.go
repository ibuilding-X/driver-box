package store

import (
	"github.com/patrickmn/go-cache"
	"time"
)

//var store *cache.Cache

type Handler struct {
	Store *cache.Cache
}

// Init init store
func Init() *Handler {
	newStore := cache.New(cache.NoExpiration, cache.DefaultExpiration)
	store := &Handler{Store: newStore}
	return store
}

// Get an item from the store. Returns the item or nil, and a bool indicating
// whether the key was found.
func (l *Handler) Get(key string) (interface{}, bool) {
	value, found := l.Store.Get(key)
	return value, found
}

// Set an item to the store, replacing any existing item. If the duration is 0
// (DefaultExpiration), the store's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (l *Handler) Set(key string, value interface{}, d time.Duration) {
	l.Store.Set(key, value, d)
}
