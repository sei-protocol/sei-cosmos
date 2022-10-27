package accesscontrol

import (
	"reflect"
	"testing"

	abci "github.com/tendermint/tendermint/abci/types"
)

func TestAccessTypeStringToEnum(t *testing.T) {
	type args struct {
		accessType string
	}
	tests := []struct {
		name string
		args args
		want AccessType
	}{
		{
			name: "read",
			args: args{accessType: "rEad"},
			want: AccessType_READ,
		},
		{
			name: "write",
			args: args{accessType: "wriTe"},
			want: AccessType_WRITE,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AccessTypeStringToEnum(tt.args.accessType); got != tt.want {
				t.Errorf("AccessTypeStringToEnum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComparator_IsConcurrentSafeIdentifier(t *testing.T) {
	type fields struct {
		AccessType AccessType
		Identifier string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "is safe",
			fields: fields{AccessType: AccessType_WRITE, Identifier: "bank/SendEnabled"},
			want:   true,
		},
		{
			name:   "not safe",
			fields: fields{AccessType: AccessType_WRITE, Identifier: "some contract addr"},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Comparator{
				AccessType: tt.fields.AccessType,
				Identifier: tt.fields.Identifier,
			}
			if got := c.IsConcurrentSafeIdentifier(); got != tt.want {
				t.Errorf("Comparator.IsConcurrentSafeIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComparator_Contains(t *testing.T) {
	type fields struct {
		AccessType AccessType
		Identifier string
	}
	type args struct {
		comparator Comparator
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "contains same type",
			fields: fields{AccessType: AccessType_WRITE, Identifier: "a/b/c/d/e"},
			args:   args{comparator: Comparator{AccessType: AccessType_WRITE, Identifier: "a/b/c/d"}},
			want:   true,
		},
		{
			name:   "contains diff type",
			fields: fields{AccessType: AccessType_WRITE, Identifier: "a/b/c/d/e"},
			args:   args{comparator: Comparator{AccessType: AccessType_READ, Identifier: "a/b/c/d"}},
			want:   false,
		},
		{
			name:   "does not contains same type",
			fields: fields{AccessType: AccessType_READ, Identifier: "a/b/c/d/e"},
			args:   args{comparator: Comparator{AccessType: AccessType_READ, Identifier: "d/a/b/c"}},
			want:   false,
		},
		{
			name:   "does not contains diff type",
			fields: fields{AccessType: AccessType_READ, Identifier: "a/b/c/d/e"},
			args:   args{comparator: Comparator{AccessType: AccessType_WRITE, Identifier: "d/a/b/c"}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Comparator{
				AccessType: tt.fields.AccessType,
				Identifier: tt.fields.Identifier,
			}
			if got := c.Contains(tt.args.comparator); got != tt.want {
				t.Errorf("Comparator.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAccessOperations(t *testing.T) {
	type args struct {
		accessOps []AccessOperation
		events    []abci.Event
	}
	tests := []struct {
		name string
		args args
		want map[Comparator]bool
	}{
		{
			name:   "empty",
			args:   args{
						accessOps: []AccessOperation{},
						events: []abci.Event{},
					},
			want:   map[Comparator]bool{},
		},
		{
			name:   "missing",
			args:   args{
						accessOps: []AccessOperation{},
						events: []abci.Event{
							{
								Type: "resource_access",
								Attributes: []abci.EventAttribute{
									{Key: "key", Value: "a/b/c/d/e"},
									{Key: "access_type", Value: "write"},
								},
							},
						},
					},
			want:   map[Comparator]bool{
				{AccessType: AccessType_WRITE, Identifier: "a/b/c/d/e"}: true,
			},
		},
		{
			name:   "extra access ops",
			args:   args{
						accessOps: []AccessOperation{
							{AccessType: AccessType_READ, IdentifierTemplate: "abc/defg", ResourceType: ResourceType_KV},
						},
						events: []abci.Event{},
					},
			want:   map[Comparator]bool{},
		},
		{
			name:   "matched",
			args:   args{
						accessOps: []AccessOperation{
							{AccessType: AccessType_WRITE, IdentifierTemplate: "abc/defg", ResourceType: ResourceType_KV},
						},
						events: []abci.Event{
							{
								Type: "resource_access",
								Attributes: []abci.EventAttribute{
									{Key: "key", Value: "abc/defg/e"},
									{Key: "access_type", Value: "write"},
								},
							},
						},
					},
			want:   map[Comparator]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateAccessOperations(tt.args.accessOps, tt.args.events); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateAccessOperations() = %v, want %v", got, tt.want)
			}
		})
	}
}
