package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/hiteshrepo/hazardous/pkg/hazardous"
)

var (
	excludeDirs       string
	allowedExtensions string
)

func main() {
	flag.StringVar(&allowedExtensions, "allow-extensions", "", "Comma-separated list of allowed file extensions")
	flag.StringVar(&excludeDirs, "exclude-dirs", "", "Comma-separated list of directories to exclude")
	flag.Parse()

	exts := strings.Split(allowedExtensions, ",")
	excluded := strings.Split(excludeDirs, ",")

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("please provide a path or path-pattern (e.g., ./..., ./*, dir/*, dir/file.sh)")
		return
	}

	if len(args) == 1 && !strings.HasSuffix(args[0], "...") {
		log.Fatal("only path or path-pattern is expected (e.g., ./..., ./*, dir/*, dir/file.sh)")
		return
	}

	if len(args) == 1 && strings.HasSuffix(args[0], "...") {
		hazardous.HandleRecursive(args[0], exts, excluded)
		return
	}

	if len(args) == 1 && strings.HasSuffix(args[0], "*") {
		hazardous.HandleGlob(args[0], exts, excluded)
		return
	}

	if len(args) == 1 {
		fInfo, err := os.Stat(args[0])
		if err != nil {
			log.Fatalf("error checking file or directory: %v", err)
			return
		}

		if fInfo.IsDir() {
			hazardous.TraverseAndHandleFile(fInfo.Name(), exts, excluded)
		} else {
			hazardous.HandleFile(fInfo.Name(), exts)
		}
	}

	if len(args) > 1 {
		for _, dir := range args {
			hazardous.HandleRecursive(dir, exts, excluded)
		}

		return
	}
}
