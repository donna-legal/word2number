package word2number

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/donna-legal/word2number/resources"
)

//go:generate go-bindata -pkg resources -o resources/resources.go -ignore=.*\.go resources

// Converter keeps the necessary information to convert words to numbers
type Converter struct {
	lang         string
	counters     []counterType
	multipliers  []counterType
	dividers     []counterType
	decimals     []decimalType
	digitPattern *regexp.Regexp
}
type decimalType struct {
	pattern *regexp.Regexp
	weak    bool
}
type counterType struct {
	value   float64
	pattern *regexp.Regexp
}

// NewConverter creates a new word2number converter
func NewConverter(locale string) (*Converter, error) {
	if !resources.HasLocale(locale) {
		return nil, errors.New("language not supported: " + locale)
	}
	c := &Converter{lang: locale}
	c.digitPattern = regexp.MustCompile(`\b\d+(\.\d+)?\b`)

	for _, m := range resources.ArrayMap(locale, "decimals") {
		pattern := regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b`, m["word"]))
		c.decimals = append(c.decimals, decimalType{pattern, m["weak"] == "true"})
	}

	for _, counter := range resources.ArrayMap(locale, "counters") {
		ct := newCounterType(counter)
		c.counters = append(c.counters, ct)
	}

	for _, multi := range resources.ArrayMap(locale, "multipliers") {
		ct := newCounterType(multi)
		c.multipliers = append(c.multipliers, ct)
	}

	for _, m := range resources.ArrayMap(locale, "dividers") {
		ct := newCounterType(m)
		c.dividers = append(c.dividers, ct)
	}
	return c, nil
}

func newCounterType(m map[string]string) (c counterType) {
	var err error
	c.pattern = regexp.MustCompile(fmt.Sprintf(`(?i)%s`, m["word"]))
	c.value, err = strconv.ParseFloat(m["number"], 64)
	if err != nil {
		panic(err)
	}
	return
}

// Words2Number takes in a string and returns a floating point
func (c *Converter) Words2Number(words string) float64 {
	ms := c.findMatches(words)
	ms.removeOverlaps()
	sort.Sort(ms)
	before, after := ms.splitOn()

	sum := getValues(before)
	decimals := getDecimals(after)

	return sum + decimals
}

func (c *Converter) findMatches(words string) matches {
	var ms matches
	for _, m := range c.digitPattern.FindAllStringIndex(words, -1) {
		d := words[m[0]:m[1]]
		n, _ := strconv.ParseFloat(d, 64) // TODO: handle this potential error
		ms = append(ms, newMatch(countKey, m, words, n))
	}
	for _, count := range c.counters {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			ms = append(ms, newMatch(countKey, m, words, count.value))
		}
	}
	for _, count := range c.multipliers {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			ms = append(ms, newMatch(multiKey, m, words, count.value))
		}
	}
	for _, d := range c.decimals {
		for _, m := range d.pattern.FindAllStringIndex(words, -1) {
			t := decimalKey
			if d.weak {
				t = weakDecimalKey
			}
			ms = append(ms, newMatch(t, m, words, 0))
		}
	}
	for _, count := range c.dividers {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			ms = append(ms, newMatch(dividerKey, m, words, count.value))
		}
	}
	return ms
}

func getValues(vals matches) (out float64) {
	var sums []float64
	for _, m := range vals {
		switch m.tyype {
		case countKey:
			sums = append([]float64{m.numeric}, sums...)
		case multiKey:
			if len(sums) == 0 {
				sums = []float64{1}
			}
			for i, s := range sums {
				if s > m.numeric {
					break
				}
				sums[i] *= m.numeric
			}
		}
	}
	for _, s := range sums {
		out += s
	}
	return

}

func getDecimals(after matches) float64 {
	divideMode := true
	divider := 1.0
	multiplier := 1.0
	dsum := 0.0
	for i := len(after) - 1; i >= 0; i-- {
		m := after[i]
		switch m.tyype {
		case dividerKey:
			divider = m.numeric
		case multiKey:
			if divideMode {
				divider *= m.numeric
			} else {
				multiplier *= m.numeric
			}

		case countKey:
			dsum += multiplier * m.numeric
			multiplier = 1
			divideMode = false
		}
	}
	decimals := dsum / divider
	for decimals >= 1 {
		decimals /= 10.0
	}
	return decimals
}
