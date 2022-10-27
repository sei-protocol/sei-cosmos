package accesscontrol

import "testing"

func TestComparator_DependencyMatch(t *testing.T) {
	type fields struct {
		AccessType AccessType
		Identifier string
		StoreKey   string
	}
	type args struct {
		accessOp AccessOperation
		prefix   []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Comparator{
				AccessType: tt.fields.AccessType,
				Identifier: tt.fields.Identifier,
				StoreKey:   tt.fields.StoreKey,
			}
			if got := c.DependencyMatch(tt.args.accessOp, tt.args.prefix); got != tt.want {
				t.Errorf("Comparator.DependencyMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
