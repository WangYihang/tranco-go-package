package tranco_test

import (
	"fmt"
	"testing"

	"github.com/WangYihang/tranco"
)

func ExampleNewTrancoList() {
	list, err := tranco.NewTrancoList("2019-04-30", false, "1000")
	if err != nil {
		panic(err)
	}
	rank, err := list.Rank("google.com")
	if err != nil {
		panic(err)
	}
	fmt.Println(rank)
	// Output: 1
}

func BenchmarkDomainLookup(b *testing.B) {
	list, err := tranco.NewTrancoList("2019-04-30", false, "1000")
	if err != nil {
		panic(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.Rank("google.com")
	}
}
