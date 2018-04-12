package word2number

import (
	"fmt"
	"testing"
)

func TestConverter_Words2Number(t *testing.T) {
	c, _ := NewConverter("en")
	tests := []struct {
		words string
		want  float64
	}{
		// Simple
		{"one", 1},
		{"two", 2},
		{"three", 3},
		{"sixteen", 16},
		// Simple multi-word
		{"one hundred", 100},
		{"one-hundred", 100},
		{"two-hundred", 200},
		{"two thousand", 2000},
		// More complicated multiword
		{"two thousand three-hundred seventy five", 2375},
		{"two hundred thousand five", 200005},
		// Decimals
		{"oh point twenty-five", 0.25},
		{"zero point five thousandths", 0.005},
		{"ten point fifty-five hundredths", 10.55},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprint("testcase-", i), func(t *testing.T) {
			if got := c.Words2Number(tt.words); got != tt.want {
				t.Errorf("Converter.Words2Number(%s) = %v, want %v", tt.words, got, tt.want)
			}
		})
	}
}
