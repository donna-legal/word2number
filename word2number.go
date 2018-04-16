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
	c.pattern = regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b`, m["word"]))
	c.value, err = strconv.ParseFloat(m["number"], 64)
	if err != nil {
		panic(err)
	}
	return
}

const (
	none = iota
	countKey
	multiKey
	dividerKey
	decimalKey
	weakDecimalKey
)

func newMatch(t int, m []int, words string, value float64) match {
	return match{
		value:   words[m[0]:m[1]],
		tyype:   t,
		numeric: value,
		start:   m[0],
		end:     m[1],
	}
}

type match struct {
	value   string
	numeric float64
	tyype   int
	start   int
	end     int
}

type matches []match

func (mas matches) Len() int {
	return len(mas)
}

func (mas matches) Less(i, j int) bool {
	return mas[i].start < mas[j].start
}

func (mas matches) Swap(i, j int) {
	mas[i], mas[j] = mas[j], mas[i]
}

func (mas matches) splitOn() (before matches, after matches) {
	var split, weakSplit, divider bool
	for _, m := range mas {
		switch m.tyype {
		case decimalKey:
			split = true
		case weakDecimalKey:
			weakSplit = true
		case dividerKey:
			divider = true
			fallthrough
		default:
			if !split && !weakSplit {
				before = append(before, m)
			} else {
				after = append(after, m)
			}
		}
	}
	if !split && !weakSplit && divider {
		return after, before
	}
	if weakSplit && !divider {
		return append(before, after...), matches{}
	}
	return
}

// Words2Number takes in a string and returns a floating point
func (c *Converter) Words2Number(words string) float64 {
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
	sort.Sort(ms)
	before, after := ms.splitOn()

	sum := getValues(before)
	decimals := getDecimals(after)

	return sum + decimals
}

func getValues(before matches) float64 {
	var lastMatch, lastMultiplier match
	lastNum := 0.0
	sum := 0.0
	for _, m := range before {
		switch m.tyype {
		case multiKey:
			if lastMultiplier.tyype == multiKey && lastMultiplier != lastMatch && lastMultiplier.numeric < m.numeric {
				sum += lastNum
				sum *= m.numeric
				lastNum = 0
			} else {
				lastNum *= m.numeric
			}
			lastMultiplier = m
		case countKey:
			if lastMatch.tyype != multiKey {
				lastNum += m.numeric
			} else {
				sum += lastNum
				lastNum = m.numeric
			}
		}
		lastMatch = m
	}
	return sum + lastNum
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
