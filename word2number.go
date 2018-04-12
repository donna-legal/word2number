package word2number

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/donna-legal/word2number/resources"
)

//go:generate go-bindata -pkg resources -o resources/resources.go -ignore=.*\.go resources

// Converter keeps the necessary information to convert words to numbers
type Converter struct {
	lang        string
	counters    []counterType
	multipliers []counterType
	dividers    []counterType
	decimals    []*regexp.Regexp
}

type counterType struct {
	value   int
	pattern *regexp.Regexp
}

// NewConverter creates a new word2number converter
func NewConverter(lang string) (*Converter, error) {
	c := &Converter{lang: lang}
	decimals := resources.ArrayMap(lang, "decimals")
	for _, m := range decimals {
		pattern, err := regexp.Compile(fmt.Sprintf(`(?i)\b%s\b`, m["word"]))
		if err != nil {
			return nil, err
		}
		c.decimals = append(c.decimals, pattern)
	}
	counters := resources.ArrayMap(lang, "counters")
	for _, counter := range counters {
		ct, err := newCounterType(counter)
		if err != nil {
			return nil, err
		}
		c.counters = append(c.counters, ct)
	}
	for _, multi := range resources.ArrayMap(lang, "multipliers") {
		ct, err := newCounterType(multi)
		if err != nil {
			return nil, err
		}
		c.multipliers = append(c.multipliers, ct)
	}
	for _, m := range resources.ArrayMap(lang, "dividers") {
		ct, err := newCounterType(m)
		if err != nil {
			return nil, err
		}
		c.dividers = append(c.dividers, ct)
	}
	return c, nil
}

func newCounterType(m map[string]string) (c counterType, err error) {
	c.pattern, err = regexp.Compile(fmt.Sprintf(`(?i)\b%s\b`, m["word"]))
	if err != nil {
		return c, err
	}
	c.value, err = strconv.Atoi(m["number"])
	return
}

const (
	countKey = iota
	multiKey
	dividerKey
	decimalKey
)

func newMatch(t int, m []int, words string, value int) match {
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
	tyype   int
	numeric int
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
	var split bool
	for _, m := range mas {
		if m.tyype == tyype {
			split = true
			continue
		}
		if split {
			before = append(before, m)
		} else {
			after = append(after, m)
		}
	}
	if !split {
		return after, before
	}
	return
}

// Words2Number takes in a string and returns a floating point
func (c *Converter) Words2Number(words string) float64 {
	var matches matches
	for _, count := range c.counters {
		ms := count.pattern.FindAllStringIndex(words, -1)
		for _, m := range ms {
			matches = append(matches, newMatch(countKey, m, words, count.value))
		}
	}
	for _, count := range c.multipliers {
		ms := count.pattern.FindAllStringIndex(words, -1)
		for _, m := range ms {
			matches = append(matches, newMatch(multiKey, m, words, count.value))
		}
	}
	for _, d := range c.decimals {
		for _, m := range d.FindAllStringIndex(words, -1) {
			matches = append(matches, newMatch(decimalKey, m, words, 0))
		}
	}
	for _, count := range c.dividers {
		ms := count.pattern.FindAllStringIndex(words, -1)
		for _, m := range ms {
			matches = append(matches, newMatch(dividerKey, m, words, count.value))
		}
	}
	sort.Sort(matches)
	before, after := matches.splitOn(decimalKey)
	sum := 0
	multiplier := 1
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
	divider := 1
	multiplier = 1
	dsum := 0
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
	decimals := float64(dsum) / float64(divider)
	for decimals >= 1 {
		decimals /= 10.0
	}
	return float64(sum) + decimals
}
