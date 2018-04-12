package resources

import (
	"fmt"
	"reflect"
	"testing"
)

func TestArray(t *testing.T) {
	tests := []struct {
		locale  string
		key     string
		wantOut []map[string]string
	}{
		{"en", "decimals", []map[string]string{map[string]string{"word": "point"}}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprint("test case ", i), func(t *testing.T) {
			if gotOut := ArrayMap(tt.locale, tt.key); !reflect.DeepEqual(gotOut, tt.wantOut) {
				t.Errorf("Array() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}
