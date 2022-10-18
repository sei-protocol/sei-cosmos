package types

import (
	"reflect"
	"sync"
	"testing"
)

func TestContextMemCache_GetSortedDeferredSendsKeys(t *testing.T) {
	type fields struct {
		deferredSends     map[string]Coins
		deferredSendsLock *sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ContextMemCache{
				deferredSends:     tt.fields.deferredSends,
				deferredSendsLock: tt.fields.deferredSendsLock,
			}
			if got := c.GetSortedDeferredSendsKeys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ContextMemCache.GetSortedDeferredSendsKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}
