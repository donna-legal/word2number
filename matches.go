package word2number

const (
	none = iota
	countKey
	multiKey
	dividerKey
	decimalKey
	weakDecimalKey
	percentKey
)

func newMatch(t int, m []int, words string, value float64, multipliable bool) match {
	return match{
		value:        words[m[0]:m[1]],
		tyype:        t,
		numeric:      value,
		start:        m[0],
		end:          m[1],
		multipliable: multipliable,
	}
}

type match struct {
	value        string
	numeric      float64
	tyype        int
	start        int
	end          int
	multipliable bool
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
			// Reset potential weak split
			weakSplit = false
			before = append(before, after...)
			after = matches{}
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

func (mas *matches) removeOverlaps() {
	for i, a := range *mas {
		for j, b := range *mas {
			if i != j {
				if a.start <= b.start && a.end >= b.end {
					*mas = append((*mas)[:j], (*mas)[j+1:]...)
					mas.removeOverlaps()
					return
				}
			}
		}
	}
}
