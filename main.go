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

	targetPath := args[0]
	if targetPath == "./..." {
		err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && shouldScanFile(path, config) {
				scanFile(path)
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if shouldScanFile(targetPath, config) {
			scanFile(targetPath)
		}
	}
}

func shouldScanFile(targetPath string, config Config) bool {
	return helpers.IsAllowedExtension(targetPath, config.allowedExtensions) &&
		!helpers.IsExcludedDir(targetPath, config.excludeDirs)
}

func scanFile(filepath string) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Printf("Error reading file %s: %v", filepath, err)
		return
	}

	if strings.HasSuffix(filepath, "Makefile") {
		scanMakefile(string(content), filepath)
	} else {
		issues := scanShellScript(string(content), filepath)
		issue.ReportIssues(issues)
	}
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
		if cmd, ok := node.(*syntax.CallExpr); ok {
			if issue := hazardous.CheckHazardousCommand(cmd, filepath); issue != nil {
				issues = append(issues, *issue)
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
		flag, col := hazardous.CheckHazardousLine(line, "rm")
		if len(flag) > 0 {
			issues = append(issues, issue.Issue{
				Filepath: filepath,
				Line:     uint(i + 1),
				Col:      col,
				Command:  "rm " + flag,
			})
		}
	}

	return issues
}
