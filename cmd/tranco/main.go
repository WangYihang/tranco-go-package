package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/WangYihang/tranco-go-package"
	"github.com/WangYihang/tranco-go-package/pkg/util"
	"github.com/WangYihang/tranco-go-package/pkg/version"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	InputFilepath     string `short:"i" long:"input-filepath" description:"input filepath" required:"true" default:"-"`
	Date              string `short:"d" long:"date" description:"date of the list" required:"true" default:"2022-01-01"`
	IncludeSubdomains bool   `long:"include-subdomains" description:"use the full (subdomain-inclusive) list instead of second-level-domains only"`
	Version           bool   `short:"v" long:"version" description:"display version"`
	CacheFolder       string `short:"c" long:"cache-folder" description:"cache folder" default:".tranco"`
}

var cliOptions = Options{}
var listDate time.Time

func init() {
	// Parse flags
	_, err := flags.ParseArgs(&cliOptions, os.Args)
	if err != nil {
		os.Exit(1)
	}
	// Display version
	if cliOptions.Version {
		fmt.Fprintln(os.Stderr, version.Tag)
		os.Exit(0)
	}
	// Parse date
	listDate, err = time.Parse("2006-01-02", cliOptions.Date)
	if err != nil {
		slog.Error("error occured while parsing date", slog.String("date", cliOptions.Date), slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func main() {
	list, err := tranco.NewTrancoList(listDate.Format("2006-01-02"), cliOptions.IncludeSubdomains, "full", cliOptions.CacheFolder)
	if err != nil {
		slog.Error("error occured while obtaining tranco list", slog.String("date", cliOptions.Date), slog.String("error", err.Error()))
		os.Exit(1)
	}
	for domain := range util.Readlines(cliOptions.InputFilepath) {
		rank, err := list.Rank(domain)
		result := map[string]interface{}{
			"domain": domain,
			"date":   listDate.Format("2006-01-02"),
		}
		switch {
		case errors.Is(err, tranco.ErrDomainNotFound):
			// A single unranked domain shouldn't abort an entire batch;
			// record it in the output and move on to the next one.
			slog.Warn("domain not found in tranco list", slog.String("domain", domain))
			result["error"] = err.Error()
		case err != nil:
			// Any other error (I/O, cache corruption, etc.) will likely
			// recur for every remaining domain, so it's still fatal.
			slog.Error("error occured while querying rank", slog.String("domain", domain), slog.String("error", err.Error()))
			os.Exit(1)
		default:
			result["rank"] = rank
		}
		data, err := json.Marshal(result)
		if err != nil {
			slog.Error("error occured while marshalling result", slog.String("result", fmt.Sprintf("%v", result)), slog.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Println(string(data))
	}
}
