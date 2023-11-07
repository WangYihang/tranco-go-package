# GoLang bindings for Tranco List

## Cli Installation

```bash
go install github.com/WangYihang/tranco/cmd/tranco@latest
```

## Cli Usage

```bash
Usage:
  tranco [OPTIONS]

Application Options:
  -i, --input-filepath=           input filepath (default: -)
  -t, --date=                     date of the list (eg: 2023-01-01) (default: 2020-01-01)
  -s, --second-level-domain-only  only check second level domain

Help Options:
  -h, --help                      Show this help message
```

```bash
$ cat input.txt                                                    
google.com
baidu.com
tsinghua.edu.cn
pku.edu.cn

$ cat input.txt | tranco -t 2023-10-10
{"domain":"google.com","rank":1,"date":"2023-10-10"}
{"domain":"baidu.com","rank":138,"date":"2023-10-10"}
{"domain":"tsinghua.edu.cn","rank":5302,"date":"2023-10-10"}
{"domain":"pku.edu.cn","rank":4338,"date":"2023-10-10"}
```

## Module Installation

```bash
go get github.com/WangYihang/tranco
```

## Module Usage

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
