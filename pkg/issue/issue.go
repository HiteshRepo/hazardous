package issue

import (
	"fmt"
	"log"
)

type Issue struct {
	Filepath string
	Line     uint
	Col      uint
	Command  string
}

func ReportIssues(issues []Issue) {
	for _, issue := range issues {
		log.Printf("unsafe code found at position %d,%d in %s",
			issue.Line, issue.Col, issue.Filepath)
	}
}

func (i Issue) String() string {
	return fmt.Sprintf("unsafe code found at position %d,%d in %s", i.Line, i.Col, i.Filepath)
}
