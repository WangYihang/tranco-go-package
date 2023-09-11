# GoLang bindings for Tranco List

## Usage

```bash
go get github.com/WangYihang/tranco
```

```golang
package main

import (
	"fmt"
	"github.com/WangYihang/tranco"
)

func main() {
	list, err := tranco.NewTrancoList("2019-04-30", false, "1000")
	if err != nil {
		panic(err)
	}
	rank, err := list.Rank("google.com")
	if err != nil {
		panic(err)
	}
	fmt.Println(rank)
}
```
