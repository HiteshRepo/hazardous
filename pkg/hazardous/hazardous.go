package hazardous

import (
	"go/ast"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alcionai/clues"
	"github.com/hiteshrepo/hazardous/pkg/helpers"

	"golang.org/x/tools/go/analysis"
	"mvdan.cc/sh/syntax"
)

var unsafeArgPaths = map[string]any{
	"/":  nil,
	"/*": nil,
}

var Analyzer = &analysis.Analyzer{
	Name: "hazardous",
	Doc:  "check for hazardous usage of commands with unvalidated variables like 'rm -rf'",
	Run:  run,
}

func TraverseAndHandleFile(basePath string, exts, excluded []string) {
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && helpers.IsExcludedDir(path, excluded) {
			return filepath.SkipDir
		}

		HandleFile(path, exts)

		return nil
	})

	if err != nil {
		log.Fatalf("error walking the path %q: %v\n", basePath, err)
	}
}

// Handle recursive pattern (./...)
func HandleRecursive(path string, exts, excluded []string) {
	basePath := strings.TrimSuffix(path, "...")
	TraverseAndHandleFile(basePath, exts, excluded)
}

// Handle wildcard pattern (dir/*)
func HandleGlob(pattern string, exts, excluded []string) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	if len(matches) == 0 {
		return
	}

	for _, match := range matches {
		fileInfo, err := os.Stat(match)
		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			TraverseAndHandleFile(fileInfo.Name(), exts, excluded)
		} else {
			HandleFile(fileInfo.Name(), exts)
		}
	}
}

func HandleFile(filePath string, exts []string) {
	if !helpers.IsAllowedExtension(filePath, exts) {
		return
	}

	switch {
	// If we plan to scan '.go' files.
	// case strings.HasSuffix(filePath, ".go"):
	// log.Printf("handling Go file: %s\n", filePath)
	// singlechecker.Main(Analyzer)

	case strings.HasSuffix(filePath, ".sh"):
		log.Printf("handling shell script: %s\n", filePath)
		handleShellAndMakeFile(filePath)

	case strings.HasSuffix(filePath, "Makefile") || strings.HasSuffix(filePath, "/Makefile"):
		log.Printf("handling Makefile: %s\n", filePath)
		handleShellAndMakeFile(filePath)

	default:
		// skip print to reduce noise
		// log.Printf("Skipping unsupported file: %s\n", filePath)
	}
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if pos, unsafe, vars := walkNodeAndInspect(file); unsafe {
			pass.Reportf(pos, "potentially unsafe 'rm -rf' command found")

			for k, v := range vars {
				if len(strings.TrimSpace(v)) == 0 {
					log.Printf("un-assigned variable '%s' found", k)
				}
			}
		}
	}

	return nil, nil
}

func walkNodeAndInspect(file *ast.File) (token.Pos, bool, map[string]string) {
	var (
		unsafe                = false
		lastDetectedUnsafePos token.Pos
	)

	if file == nil {
		return token.NoPos, false, nil
	}

	vars := helpers.TrackVariableAssignments(file)

	ast.Inspect(file, func(n ast.Node) bool {
		if cmd, ok := n.(*ast.CallExpr); ok {
			if err := isUnsafeRMCommand(cmd, vars); err != nil {
				unsafe = true
				lastDetectedUnsafePos = cmd.Pos()
			}
		}

		return true
	})

	return lastDetectedUnsafePos, unsafe, vars
}

func isUnsafeRFFlag(args []ast.Expr, vars map[string]string) error {
	for _, arg := range args {
		if strArg, ok := arg.(*ast.Ident); ok {
			argValue := strings.TrimSpace(strArg.Name)

			if strings.Contains(argValue, "$(") || strings.Contains(argValue, "${") {
				varName, rest := helpers.ExtractVarName(argValue)

				value, ok := vars[varName]

				if !ok {
					if _, ok := unsafeArgPaths[rest]; ok {
						return clues.New("located unsafe path used, consider using ./ instead").With("PATH", argValue)
					}

					return nil
				}

				value = strings.TrimSpace(value)

				_, ok = unsafeArgPaths[value]
				if ok {
					return clues.New("unsafe value for variable, consider using ./ instead").With("VAR_NAME", varName)
				}

				if _, ok := unsafeArgPaths[value+rest]; ok {
					return clues.New("located unsafe path used, consider using ./ instead").With("PATH", argValue)
				}

				return nil
			}

			if _, ok := unsafeArgPaths[argValue]; ok {
				return clues.New("located unsafe path used, consider using ./ instead").With("PATH", argValue)
			}
		}
	}

	return nil
}

// isUnsafeRMCommand checks if the command is a potentially hazardous `rm -rf`
// and contains empty or unvalidated variables in the arguments.
func isUnsafeRMCommand(expr *ast.CallExpr, vars map[string]string) error {
	fun, ok := expr.Fun.(*ast.Ident)
	if ok && fun.Name != "rm" {
		return nil
	}

	args := expr.Args

	if len(args) > 0 {
		rfArg, ok := args[0].(*ast.Ident)
		if !ok {
			return nil
		}

		if strings.TrimSpace(rfArg.Name) != "-rf" {
			return nil
		}

		if len(args) > 1 {
			return isUnsafeRFFlag(args[1:], vars)
		}
	}

	return nil
}

func processShellScript(filePath string) (*ast.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, clues.Wrap(err, "failed to open file").With("FILE_NAME", filePath)
	}

	defer file.Close()

	parser := syntax.NewParser()
	fileNode, err := parser.Parse(file, "")
	if err != nil {
		return nil, clues.Wrap(err, "failed to parse file").With("FILE_NAME", filePath)
	}

	goAst := helpers.ConvertSyntaxNodeToAstFile(fileNode)

	// ast.Print(token.NewFileSet(), goAst)

	return goAst, nil
}

func handleShellAndMakeFile(filePath string) {
	f, err := processShellScript(filePath)
	if err != nil {
		log.Fatalf("error processing file: %v", err)
		return
	}

	tokenPos, unsafe, vars := walkNodeAndInspect(f)
	if unsafe {
		log.Printf("unsafe code found at position %d in %s\n", tokenPos, filePath)
	}

	for k, v := range vars {
		if len(strings.TrimSpace(v)) == 0 {
			log.Printf("un-assigned variable '%s' found", k)
		}
	}
}
