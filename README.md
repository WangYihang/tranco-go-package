# GoLang bindings for Tranco List

This is a GoLang bindings for [Tranco List](https://tranco-list.eu/). It can be used as a cli tool or a GoLang module.

## Cli Installation

```bash
go install github.com/WangYihang/tranco-go-package/cmd/tranco
```

## Cli Usage

```bash
Usage:
  tranco [OPTIONS]

Application Options:
  -i, --input-filepath=           input filepath that contains the domains to be queried (default: -)
  -t, --date=                     date of the list (eg: 2023-01-01) (default: 2020-01-01)
  -s, --second-level-domain-only  whether to use the list of second-level domains (default: false)

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

## Server Installation

```bash
go install github.com/WangYihang/tranco-go-package/cmd/tranco-server@latest
```

## Module Installation

```bash
go get github.com/WangYihang/tranco-go-package
```

## Module Usage

```golang
package main

import (
	"fmt"
	tranco "github.com/WangYihang/tranco-go-package"
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

## Acknowledgement

Special thanks to the authors of the [Tranco List](https://tranco-list.eu/) and the [tranco-python-package](https://github.com/DistriNet/tranco-python-package) for their work, which inspired the creation of this GoLang project.