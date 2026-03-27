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
	rmCommand = HazardousCommand{
		command: "rm",
		flags: []string{
			"-r", "--recursive",
			"-f", "--force",
			"-rf", "-fr", "--recursive --force",
		},
	}
)

// argValue extracts the string value from a shell AST Word node.
func argValue(word *syntax.Word) string {
	if len(word.Parts) == 0 {
		return ""
	}
	switch part := word.Parts[0].(type) {
	case *syntax.DblQuoted:
		if len(part.Parts) > 0 {
			return wordValue(part.Parts[0])
		}
		return ""
	default:
		return wordValue(part)
	}
}

// CheckHazardousCommand checks a shell AST CallExpr for any hazardous command pattern.
func CheckHazardousCommand(cmd *syntax.CallExpr, filepath string) *issue.Issue {
	if cmd == nil {
		return nil
	}

	cmdName := extractCommandName(cmd)
	if cmdName == "" {
		return nil
	}

	switch {
	case cmdName == rmCommand.command:
		if hasHazardousFlags(cmd, rmCommand.flags) {
			pos := cmd.Pos()
			return &issue.Issue{
				Filepath: filepath,
				Line:     pos.Line(),
				Col:      pos.Col(),
				Command:  "rm -rf",
			}
		}
	case cmdName == "chmod":
		return checkChmod(cmd, filepath)
	case cmdName == "dd":
		return checkDD(cmd, filepath)
	case cmdName == "mkfs" || strings.HasPrefix(cmdName, "mkfs."):
		return checkMkfs(cmd, filepath)
	case cmdName == "fdisk":
		return checkFdisk(cmd, filepath)
	}

	return nil
}

// CheckHazardousPipe detects dangerous pipe patterns like: curl url | bash, wget url | sh.
func CheckHazardousPipe(cmd *syntax.BinaryCmd, filepath string) *issue.Issue {
	if cmd.Op != syntax.Pipe {
		return nil
	}

	leftExpr, ok := cmd.X.Cmd.(*syntax.CallExpr)
	if !ok {
		return nil
	}
	leftName := extractCommandName(leftExpr)

	rightExpr, ok := cmd.Y.Cmd.(*syntax.CallExpr)
	if !ok {
		return nil
	}
	rightName := extractCommandName(rightExpr)

	downloadTools := map[string]bool{"curl": true, "wget": true}
	shells := map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true, "fish": true}

	if downloadTools[leftName] && shells[rightName] {
		pos := cmd.X.Pos()
		return &issue.Issue{
			Filepath: filepath,
			Line:     pos.Line(),
			Col:      pos.Col(),
			Command:  leftName + " | " + rightName,
		}
	}

	return nil
}

// CheckHazardousLine checks a single Makefile line for the rm command's hazardous flags.
// Returns the detected flag string and column position, or empty string and 0 if no issue.
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

// CheckMakefileLine checks a Makefile line for all hazardous command patterns.
// Returns all issues found on this line.
func CheckMakefileLine(line, filepath string, lineNum uint) []issue.Issue {
	var issues []issue.Issue

	if flag, col := CheckHazardousLine(line, "rm"); len(flag) > 0 {
		issues = append(issues, issue.Issue{
			Filepath: filepath,
			Line:     lineNum,
			Col:      col,
			Command:  "rm " + flag,
		})
	}

	if cmd, col := chmodMakefileLine(line); len(cmd) > 0 {
		issues = append(issues, issue.Issue{
			Filepath: filepath,
			Line:     lineNum,
			Col:      col,
			Command:  cmd,
		})
	}

	if cmd, col := ddMakefileLine(line); len(cmd) > 0 {
		issues = append(issues, issue.Issue{
			Filepath: filepath,
			Line:     lineNum,
			Col:      col,
			Command:  cmd,
		})
	}

	if cmd, col := mkfsMakefileLine(line); len(cmd) > 0 {
		issues = append(issues, issue.Issue{
			Filepath: filepath,
			Line:     lineNum,
			Col:      col,
			Command:  cmd,
		})
	}

	if cmd, col := fdiskMakefileLine(line); len(cmd) > 0 {
		issues = append(issues, issue.Issue{
			Filepath: filepath,
			Line:     lineNum,
			Col:      col,
			Command:  cmd,
		})
	}

	if cmd, col := pipeMakefileLine(line); len(cmd) > 0 {
		issues = append(issues, issue.Issue{
			Filepath: filepath,
			Line:     lineNum,
			Col:      col,
			Command:  cmd,
		})
	}

	return issues
}

// --- Shell AST checkers ---

func checkChmod(cmd *syntax.CallExpr, filepath string) *issue.Issue {
	for _, word := range cmd.Args[1:] {
		val := argValue(word)
		if isChmodRecursive(val) || isDangerousPermission(val) {
			pos := cmd.Pos()
			return &issue.Issue{
				Filepath: filepath,
				Line:     pos.Line(),
				Col:      pos.Col(),
				Command:  "chmod " + val,
			}
		}
	}
	return nil
}

func checkDD(cmd *syntax.CallExpr, filepath string) *issue.Issue {
	for _, word := range cmd.Args[1:] {
		val := argValue(word)
		if strings.HasPrefix(val, "of=/dev/") {
			pos := cmd.Pos()
			return &issue.Issue{
				Filepath: filepath,
				Line:     pos.Line(),
				Col:      pos.Col(),
				Command:  "dd " + val,
			}
		}
	}
	return nil
}

func checkMkfs(cmd *syntax.CallExpr, filepath string) *issue.Issue {
	cmdName := extractCommandName(cmd)
	pos := cmd.Pos()
	return &issue.Issue{
		Filepath: filepath,
		Line:     pos.Line(),
		Col:      pos.Col(),
		Command:  cmdName,
	}
}

func checkFdisk(cmd *syntax.CallExpr, filepath string) *issue.Issue {
	for _, word := range cmd.Args[1:] {
		val := argValue(word)
		if strings.HasPrefix(val, "/dev/") {
			pos := cmd.Pos()
			return &issue.Issue{
				Filepath: filepath,
				Line:     pos.Line(),
				Col:      pos.Col(),
				Command:  "fdisk " + val,
			}
		}
	}
	return nil
}

// isChmodRecursive returns true if the flag string represents a recursive chmod flag.
// Handles -R, --recursive, and combined short flags like -Rv, -vR.
func isChmodRecursive(val string) bool {
	if val == "--recursive" {
		return true
	}
	if strings.HasPrefix(val, "-") && !strings.HasPrefix(val, "--") {
		return strings.ContainsRune(val, 'R')
	}
	return false
}

// isDangerousPermission returns true for world-writable or fully-open permission modes.
func isDangerousPermission(perm string) bool {
	dangerous := map[string]bool{
		"777": true, "0777": true,
		"666": true, "0666": true,
		"a+w": true, "o+w": true, "+w": true,
		"a+rwx": true, "+rwx": true,
	}
	return dangerous[perm]
}

// --- Makefile line checkers ---

// makeCmd strips Makefile execution prefixes (@, -, +) from a command token.
// These prefixes are attached to command names only, not to arguments.
func makeCmd(word string) string {
	return strings.TrimLeft(strings.TrimSpace(word), "@+-")
}

func chmodMakefileLine(line string) (string, uint) {
	words := splitCommand(line)
	found := false
	for _, word := range words {
		if !found {
			if strings.EqualFold(makeCmd(word), "chmod") {
				found = true
			}
			continue
		}
		// word is an argument — do not strip prefixes
		arg := strings.TrimSpace(word)
		if isChmodRecursive(arg) || isDangerousPermission(arg) {
			return "chmod " + arg, colOf(line, "chmod")
		}
	}
	return "", 0
}

func ddMakefileLine(line string) (string, uint) {
	words := splitCommand(line)
	found := false
	for _, word := range words {
		if !found {
			if strings.EqualFold(makeCmd(word), "dd") {
				found = true
			}
			continue
		}
		arg := strings.TrimSpace(word)
		if strings.HasPrefix(arg, "of=/dev/") {
			return "dd " + arg, colOf(line, "dd")
		}
	}
	return "", 0
}

func mkfsMakefileLine(line string) (string, uint) {
	words := splitCommand(line)
	for _, word := range words {
		w := makeCmd(word)
		if w == "mkfs" || strings.HasPrefix(w, "mkfs.") {
			return w, colOf(line, w)
		}
	}
	return "", 0
}

func fdiskMakefileLine(line string) (string, uint) {
	words := splitCommand(line)
	found := false
	for _, word := range words {
		if !found {
			if strings.EqualFold(makeCmd(word), "fdisk") {
				found = true
			}
			continue
		}
		arg := strings.TrimSpace(word)
		if strings.HasPrefix(arg, "/dev/") {
			return "fdisk " + arg, colOf(line, "fdisk")
		}
	}
	return "", 0
}

func pipeMakefileLine(line string) (string, uint) {
	pipeIdx := strings.Index(line, "|")
	if pipeIdx < 0 {
		return "", 0
	}

	leftWords := splitCommand(strings.TrimSpace(line[:pipeIdx]))
	rightWords := splitCommand(strings.TrimSpace(line[pipeIdx+1:]))

	if len(leftWords) == 0 || len(rightWords) == 0 {
		return "", 0
	}

	downloadTools := map[string]bool{"curl": true, "wget": true}
	shells := map[string]bool{"bash": true, "sh": true, "zsh": true, "dash": true, "fish": true}

	leftCmd := makeCmd(leftWords[0])
	rightCmd := strings.TrimSpace(rightWords[0])

	if downloadTools[leftCmd] && shells[rightCmd] {
		return leftCmd + " | " + rightCmd, colOf(line, leftCmd)
	}
	return "", 0
}

// colOf returns the 1-based column position of needle in line.
func colOf(line, needle string) uint {
	idx := strings.Index(line, needle)
	if idx < 0 {
		return 1
	}
	return uint(idx + 1)
}

// --- Existing unexported helpers ---

func splitCommand(input string) []string {
	re := regexp.MustCompile(`\s+`)
	return re.Split(strings.TrimSpace(input), -1)
}

func wordValue(wordPart syntax.WordPart) string {
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
		value := argValue(word)

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
