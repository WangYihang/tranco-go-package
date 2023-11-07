package util

import (
	"bufio"
	"log/slog"
	"os"
	"strings"
)

func Readlines(filePath string) chan string {
	out := make(chan string)
	go func() {
		defer close(out)

		var fd *os.File
		var err error
		if filePath == "-" {
			fd = os.Stdin
		} else {
			fd, err = os.OpenFile(filePath, os.O_RDONLY, 0)
			if err != nil {
				slog.Error("error occured while opening file", slog.String("file", filePath), slog.String("error", err.Error()))
				return
			}
		}
		defer fd.Close()

		scanner := bufio.NewScanner(fd)
		kiloByte := 1024
		megaByte := 1024 * kiloByte
		gigaByte := 1024 * megaByte
		buf := make([]byte, 0, megaByte)
		scanner.Buffer(buf, gigaByte)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				out <- line
			}
		}
		if err := scanner.Err(); err != nil {
			slog.Error("error occurs while scanning lines", slog.String("error", err.Error()))
			return
		}
	}()
	return out
}
