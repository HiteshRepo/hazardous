package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hiteshrepo/hazardous/pkg/hazardous"
	"github.com/hiteshrepo/hazardous/pkg/helpers"
	"github.com/hiteshrepo/hazardous/pkg/issue"

	"mvdan.cc/sh/syntax"
)

type Config struct {
	allowedExtensions []string
	excludeDirs       []string
}

func main() {
	os.Exit(run())
}

func run() int {
	extensions := flag.String("allow-extensions", ".sh,Makefile", "Comma-separated list of allowed file extensions")
	excludes := flag.String("exclude-dirs", "node_modules,linters", "Comma-separated list of directories to exclude")
	flag.Parse()

	config := Config{
		allowedExtensions: strings.Split(*extensions, ","),
		excludeDirs:       strings.Split(*excludes, ","),
	}

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("Please provide a path to scan")
	}

	var allIssues []issue.Issue
	targetPath := args[0]

	if targetPath == "./..." {
		err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && shouldScanFile(path, config) {
				allIssues = append(allIssues, scanFile(path)...)
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if shouldScanFile(targetPath, config) {
			allIssues = scanFile(targetPath)
		}
	}

	issue.ReportIssues(allIssues)
	if len(allIssues) > 0 {
		return 1
	}
	return 0
}

func shouldScanFile(targetPath string, config Config) bool {
	return helpers.IsAllowedExtension(targetPath, config.allowedExtensions) &&
		!helpers.IsExcludedDir(targetPath, config.excludeDirs)
}

func scanFile(filepath string) []issue.Issue {
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Printf("Error reading file %s: %v", filepath, err)
		return nil
	}

	if strings.HasSuffix(filepath, "Makefile") {
		return scanMakefile(string(content), filepath)
	}
	return scanShellScript(string(content), filepath)
}

func scanShellScript(content, filepath string) []issue.Issue {
	reader := strings.NewReader(content)
	file, err := syntax.NewParser().Parse(reader, filepath)
	if err != nil {
		log.Printf("Error parsing file %s: %v", filepath, err)
		return nil
	}

	var issues []issue.Issue
	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CallExpr:
			if iss := hazardous.CheckHazardousCommand(n, filepath); iss != nil {
				issues = append(issues, *iss)
			}
		case *syntax.BinaryCmd:
			if iss := hazardous.CheckHazardousPipe(n, filepath); iss != nil {
				issues = append(issues, *iss)
			}
		}
		return true
	})

	return issues
}

func scanMakefile(content, filepath string) []issue.Issue {
	var issues []issue.Issue
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineIssues := hazardous.CheckMakefileLine(line, filepath, uint(i+1))
		issues = append(issues, lineIssues...)
	}

	return issues
}
