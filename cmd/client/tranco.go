package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/WangYihang/tranco"
	"github.com/WangYihang/tranco-go-package/pkg/util"
	"github.com/WangYihang/tranco-go-package/pkg/version"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	InputFilepath         string `short:"i" long:"input-filepath" description:"input filepath" required:"true" default:"-"`
	Date                  string `short:"d" long:"date" description:"date of the list" required:"true" default:"2022-01-01"`
	SecondLevelDomainOnly bool   `short:"s" long:"second-level-domain-only" description:"only check second level domain"`
}

var cliOptions = Options{}
var listDate time.Time

func init() {
	slog.Info("tranco-go", slog.String("version", version.PV.Short()))
	// Parse flags
	_, err := flags.ParseArgs(&cliOptions, os.Args)
	if err != nil {
		os.Exit(1)
	}
	// Parse date
	listDate, err = time.Parse("2006-01-02", cliOptions.Date)
	if err != nil {
		slog.Error("error occured while parsing date", slog.String("date", cliOptions.Date), slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func main() {
	list, err := tranco.NewTrancoList(listDate.Format("2006-01-02"), !cliOptions.SecondLevelDomainOnly, "full")
	if err != nil {
		slog.Error("error occured while parsing date", slog.String("date", cliOptions.Date), slog.String("error", err.Error()))
		os.Exit(1)
	}
	for domain := range util.Readlines(cliOptions.InputFilepath) {
		rank, err := list.Rank(domain)
		if err != nil {
			slog.Error("error occured while querying rank", slog.String("domain", domain), slog.String("error", err.Error()))
			os.Exit(1)
		}
		result := map[string]interface{}{
			"domain": domain,
			"rank":   rank,
			"date":   listDate.Format("2006-01-02"),
		}
		data, err := json.Marshal(result)
		if err != nil {
			slog.Error("error occured while marshalling result", slog.String("result", fmt.Sprintf("%v", result)), slog.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Println(string(data))
	}
}
