package word2number

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/donna-legal/word2number/resources"
)

//go:generate go-bindata -pkg resources -o resources/resources.go -ignore=.*\.go resources

// Converter keeps the necessary information to convert words to numbers
type Converter struct {
	lang         string
	counters     []counterType
	multipliers  []counterType
	dividers     []counterType
	percents     []counterType
	decimals     []decimalType
	digitPattern *regexp.Regexp
	words        map[int]string
}
type decimalType struct {
	pattern *regexp.Regexp
	weak    bool
}
type counterType struct {
	value        float64
	multipliable bool
	pattern      *regexp.Regexp
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
	c.words = make(map[int]string)
	for _, counter := range resources.ArrayMap(locale, "counters") {
		c.addToWords(counter)
		ct := newCounterType(counter)
		c.counters = append(c.counters, ct)
	}

	for _, multi := range resources.ArrayMap(locale, "multipliers") {
		c.addToWords(multi)
		ct := newCounterType(multi)
		c.multipliers = append(c.multipliers, ct)
	}

	for _, m := range resources.ArrayMap(locale, "dividers") {
		ct := newCounterType(m)
		ct.multipliable = true
		if m["multipliable"] == "false" {
			ct.multipliable = false
		}
		c.dividers = append(c.dividers, ct)
	}
	for _, m := range resources.ArrayMap(locale, "percent") {
		ct := newCounterType(m)
		ct.multipliable = true
		c.percents = append(c.percents, ct)
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

func (c *Converter) addToWords(m map[string]string) {
	i, err := strconv.Atoi(m["number"])
	if err != nil {
		panic(err)
	}
	c.words[i] = m["word"]
}

// Number2Words takes a number and returns the words for the given number
func (c *Converter) Number2Words(number float64, decimals int) (string, string) {
	var words []string
	before := int(number)
	after := number - float64(before)
	_ = after
	groups := getGroups(before)
	for i, g := range groups {
		words = append(words, c.groupToWords(g)...)
		if p := powerOf(1000, len(groups)-i); p > 1 && g > 0 {
			words = append(words, c.words[p])
		}
	}

	var afterWords []string
	groups = getGroups(int(after * float64(powerOf(10, decimals+1))))
	for i, g := range groups {
		afterWords = append(afterWords, c.groupToWords(g)...)
		if p := powerOf(1000, len(groups)-i); p > 1 && g > 0 {
			afterWords = append(afterWords, c.words[p])
		}
	}
	return strings.Join(words, " "), strings.Join(afterWords, " ")
}

func powerOf(base, k int) int {
	o := 1
	for i := 1; i < k; i++ {
		o *= base
	}
	return o
}

func getGroups(num int) (out []int) {
	for num > 0 {
		out = append([]int{num % 1000}, out...)
		num /= 1000
	}
	return
}

func (c *Converter) groupToWords(g int) (out []string) {
	if g >= 100 {
		out = append(out, c.words[g/100])
		out = append(out, c.words[100])
		g %= 100
	}
	if g >= 20 {
		out = append(out, c.words[(g/10)*10])
		g %= 10
	}
	if g > 0 {
		out = append(out, c.words[g])
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

	return (sum + decimals) / getPercent(ms)
}

func (c *Converter) findMatches(words string) matches {
	var ms matches
	for _, m := range c.digitPattern.FindAllStringIndex(words, -1) {
		d := words[m[0]:m[1]]
		n, _ := strconv.ParseFloat(d, 64) // TODO: handle this potential error
		ms = append(ms, newMatch(countKey, m, words, n, true))
	}
	for _, count := range c.counters {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			ms = append(ms, newMatch(countKey, m, words, count.value, true))
		}
	}
	for _, count := range c.multipliers {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			ms = append(ms, newMatch(multiKey, m, words, count.value, true))
		}
	}
	for _, d := range c.decimals {
		for _, m := range d.pattern.FindAllStringIndex(words, -1) {
			t := decimalKey
			if d.weak {
				t = weakDecimalKey
			}
			ms = append(ms, newMatch(t, m, words, 0, true))
		}
	}
	for _, count := range c.dividers {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			ms = append(ms, newMatch(dividerKey, m, words, count.value, count.multipliable))
		}
	}
	for _, count := range c.percents {
		for _, m := range count.pattern.FindAllStringIndex(words, -1) {
			ms = append(ms, newMatch(percentKey, m, words, count.value, count.multipliable))
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
	hasDivided := false
	divideMode := true
	divider := 1.0
	multiplier := 1.0
	dsum := 0.0
	multipliable := false
	for i := len(after) - 1; i >= 0; i-- {
		m := after[i]
		switch m.tyype {
		case dividerKey:
			divider = m.numeric
			multipliable = m.multipliable || multipliable
			hasDivided = true
		case multiKey:
			if divideMode && multipliable {
				divider *= m.numeric
			} else {
				multiplier *= m.numeric
			}
		case countKey:
			dsum += multiplier * m.numeric
			multiplier = 1
			divideMode = false
			multipliable = false
		}
	}
	if multiplier > 1 {
		dsum += multiplier
	}
	decimals := dsum / divider
	for !hasDivided && decimals >= 1 {
		decimals /= 10.0
	}
	return decimals
}

func getPercent(ms matches) float64 {
	for _, m := range ms {
		if m.tyype == percentKey {
			return m.numeric
		}
	}
	return 1
}
