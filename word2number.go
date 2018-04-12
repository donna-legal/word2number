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
	decimals     []*regexp.Regexp
	digitPattern *regexp.Regexp
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
		c.decimals = append(c.decimals, pattern)
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
	countKey = iota
	multiKey
	dividerKey
	decimalKey
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
	return mas[i].start > mas[j].start
}

func (mas matches) Swap(i, j int) {
	mas[i], mas[j] = mas[j], mas[i]
}

func (mas matches) splitOn(tyype int) (before matches, after matches) {
	var split, divider bool
	for _, m := range mas {
		switch m.tyype {
		case dividerKey:
			divider = true
			after = append(after, m)
		case decimalKey:
			split = true
		default:
			if split {
				before = append(before, m)
			} else {
				after = append(after, m)
			}
		}
	}
	if !split && !divider {
		return after, before
	}
	return
}

// Words2Number takes in a string and returns a floating point
func (c *Converter) Words2Number(words string) float64 {
	var matches matches
	for _, m := range c.digitPattern.FindAllStringIndex(words, -1) {
		d := words[m[0]:m[1]]
		n, _ := strconv.ParseFloat(d, 64) // TODO: handle this potential error
		matches = append(matches, newMatch(countKey, m, words, n))
	}
	for _, count := range c.counters {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			matches = append(matches, newMatch(countKey, m, words, count.value))
		}
	}
	for _, count := range c.multipliers {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			matches = append(matches, newMatch(multiKey, m, words, count.value))
		}
	}
	for _, d := range c.decimals {
		for _, m := range d.FindAllStringIndex(words, -1) {
			matches = append(matches, newMatch(decimalKey, m, words, 0))
		}
	}
	for _, count := range c.dividers {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			matches = append(matches, newMatch(dividerKey, m, words, count.value))
		}
	}
	sort.Sort(matches)
	before, after := matches.splitOn(decimalKey)
	sum := 0.0
	multiplier := 1.0
	for _, m := range before {
		switch m.tyype {
		case multiKey:
			multiplier *= m.numeric
		case countKey:
			sum += multiplier * m.numeric
			multiplier = 1
		}
	}
	divideMode := true
	divider := 1.0
	multiplier = 1.0
	dsum := 0.0
	for _, m := range after {
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
	return sum + decimals
}
