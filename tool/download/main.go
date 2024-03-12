package main

import (
	"log/slog"
	"os"
	"time"

	tranco "github.com/WangYihang/tranco-go-package"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Date              string `short:"d" long:"date" description:"date of the list" required:"true"`
	IncludeSubdomains bool   `short:"s" long:"include-subdomains" description:"include subdomains"`
}

var cliOptions = Options{}
var listDate time.Time

func init() {
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
	_, err := tranco.NewTrancoList(listDate.Format("2006-01-02"), cliOptions.IncludeSubdomains, "full", ".tranco")
	if err != nil {
		slog.Error("error occured while parsing date", slog.String("date", cliOptions.Date), slog.String("error", err.Error()))
		os.Exit(1)
	}
}
