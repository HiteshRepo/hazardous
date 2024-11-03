package hazardous

import (
	"regexp"
	"strings"

	"github.com/hiteshrepo/hazardous/pkg/issue"
	"mvdan.cc/sh/syntax"
)

type HazardousCommand struct {
	command string
	flags   []string
}

var (
	// Define known hazardous commands
	rmCommand = HazardousCommand{
		command: "rm",
		flags: []string{
			"-r", "--recursive",
			"-f", "--force",
			"-rf", "-fr", "--recursive --force",
		},
	}
)

func CheckHazardousCommand(cmd *syntax.CallExpr, filepath string) *issue.Issue {
	if cmd == nil {
		return nil
	}

	cmdName := extractCommandName(cmd)
	if cmdName == "" {
		return nil
	}

	switch cmdName {
	case rmCommand.command:
		if hasHazardousFlags(cmd, rmCommand.flags) {
			pos := cmd.Pos()

			return &issue.Issue{
				Filepath: filepath,
				Line:     pos.Line(),
				Col:      pos.Col(),
				Command:  "rm -rf",
			}
		}
	}

	return nil
}

func CheckHazardousLine(line, command string) (string, uint) {
	flagTillNow := ""

	words := splitCommand(line)
	for _, word := range words {
		if len(strings.TrimSpace(word)) == 0 ||
			strings.EqualFold(strings.TrimSpace(word), strings.TrimSpace(command)) {
			continue
		}

		if len(flagTillNow) == 0 {
			flagTillNow += word
		}

		if len(flagTillNow) > 0 {
			if strings.Contains(word, "--") {
				flagTillNow += " " + word
			}

			if strings.Contains(word, "-") {
				flagTillNow += strings.TrimPrefix(word, "-")
			}
		}

		for _, flag := range rmCommand.flags {
			if flagTillNow == flag {
				return flagTillNow, uint(strings.Index(line, command) + 1)
			}

			if word == flag {
				return word, uint(strings.Index(line, command) + 1)
			}
		}
	}

	return "", 0
}

func splitCommand(input string) []string {
	re := regexp.MustCompile(`\s+`)

	parts := re.Split(strings.TrimSpace(input), -1)
	return parts
}

func wordValue(wordPart syntax.WordPart) string {
	// ast.Print(token.NewFileSet(), wordPart)

	switch part := wordPart.(type) {
	case *syntax.Lit:
		return part.Value

	case *syntax.SglQuoted:
		return part.Value

	default:
		return ""
	}
}

func extractCommandName(cmd *syntax.CallExpr) string {
	if len(cmd.Args) == 0 {
		return ""
	}

	word := cmd.Args[0]

	switch part := word.Parts[0].(type) {
	case *syntax.DblQuoted:
		return wordValue(part.Parts[0])

	default:
		return wordValue(part)
	}
}

func hasHazardousFlags(cmd *syntax.CallExpr, hazardousFlags []string) bool {
	flagTillNow := ""

	for _, word := range cmd.Args[1:] {
		// ast.Print(token.NewFileSet(), word)

		value := ""
		switch part := word.Parts[0].(type) {
		case *syntax.DblQuoted:
			value = wordValue(part.Parts[0])

		default:
			value = wordValue(part)
		}

		if len(flagTillNow) == 0 {
			flagTillNow += value
		}

		if len(flagTillNow) > 0 {
			if strings.Contains(value, "--") {
				flagTillNow += " " + value
			}

			if strings.Contains(value, "-") {
				flagTillNow += strings.TrimPrefix(value, "-")
			}
		}

		for _, flag := range hazardousFlags {
			if value == flag || flagTillNow == flag {
				return true
			}
		}
	}

	return false
}
