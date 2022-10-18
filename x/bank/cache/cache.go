package cache

import (
	"context"
	"sync"
)

type BankMemCacheContextKeyType string

const BankMemStateKeyKey BankMemCacheContextKeyType = BankMemCacheContextKeyType("bank-mem-cache")

type MemState struct {
	deferredSends *sync.Map
}

func GetMemState(ctx context.Context) *MemState {
	if val := ctx.Value(BankMemStateKeyKey); val != nil {
		return val.(*MemState)
	}
	panic("cannot find mem state in context")
}

func NewMemState() *MemState {
	return &MemState{
		deferredSends: &sync.Map{},
	}
}
