# word2number

This is a library to convert words (three hundred-thousand) to numbers (300,000) in Go

## Installation

Go get the package:

`go get github.com/donna-legal/word2number`

and add it as an import:

```golang
import "github.com/donna-legal/word2number"
```

## Usage

```golang
package main

import (
    "fmt"
    "github.com/donna-legal/word2number"
)

func main() {
    converter, err := word2number.NewConverter("en")
    if err != nil {
        panic(err)
    }
    var f float64
    f = converter.Words2Number("two-thousand seventy-five")
    fmt.Println(f) // should return 2075
    f = converter.Words2Number("one-million two hundred thousand")
    fmt.Println(f) // should return 1200000
}
```

## Now and the future

Look in the test cases what works and what doesn't. 
Most things to the left of the decimal point should work.

Needs improvement:

* Decimal numbers. The simpler cases work just fine (eg. _one point three hundredths_ = 1.03), but there are quite a few failing test cases
* Things like _One point two billion_ doesn't check out at the moment either. 
