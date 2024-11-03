package helpers

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/syntax"
)

func TestTrackVariableAssignments_EmptyNode(t *testing.T) {
	node := (*ast.File)(nil)
	expected := make(map[string]string)
	actual := TrackVariableAssignments(node)

	assert.Equal(t, expected, actual)
}

func TestTrackVariableAssignments_DifferentOperators(t *testing.T) {

	table := []struct {
		name     string
		code     string
		expected map[string]string
		isGoCode bool
	}{
		{
			name: "go code",
			code: `
				package main

				func main() {
					x := 10
					y = 20
					z := "hello"
					var w int
					w = 30
				}
			`,
			expected: map[string]string{
				"x": "10",
				"y": "20",
				"z": "hello",
				"w": "30",
			},
			isGoCode: true,
		},
		{
			name: "shell script",
			code: `
		        #!/bin/bash
		        x=10
		        y=20
		        z="hello"
		        w=30
		    `,
			expected: map[string]string{
				"x": "10",
				"y": "20",
				"z": "hello",
				"w": "30",
			},
		},
		{
			name: "makefile",
			code: `
		        x := 10
		        y := 20
		        z := "hello"
		        w := 30
		    `,
			expected: map[string]string{
				"x": "10",
				"y": "20",
				"z": "hello",
				"w": "30",
			},
		},
	}

	for _, test := range table {
		log.Printf("running test %s", test.name)

		var actual map[string]string

		if test.isGoCode {
			actual = TrackVariableAssignments(goCodeToAstFile(t, test.code))
		} else {
			actual = TrackVariableAssignments(shellOrMakefileCodeToAstFile(t, test.code))
		}

		assert.Equal(t, test.expected, actual)
	}
}

func TestTrackVariableAssignments_NestedBlocks(t *testing.T) {
	code := `
		package main

		func main() {
			x := 10
			if true {
				y := 20
				z := "hello"
				var w int
				w = 30
			}
		}
	`

	expected := map[string]string{
		"x": "10",
		"y": "20",
		"z": "hello",
		"w": "30",
	}

	actual := TrackVariableAssignments(goCodeToAstFile(t, code))
	assert.Equal(t, expected, actual)
}

func TestTrackVariableAssignments_FunctionCalls(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		isGoCode bool
		expected map[string]string
	}{
		{
			name: "Go code",
			code: `
				package main

				func main() {
					x := 10
					y := calculate(20)
					z := "hello"
					var w int
					w = getResult()
				}

				func calculate(a int) int {
					return a * 2
				}

				func getResult() int {
					return 40
				}
			`,
			isGoCode: true,
			expected: map[string]string{
				"x": "10",
				"y": "calculate(20)",
				"z": "hello",
				"w": "getResult()",
			},
		},
		{
			name: "Shell script",
			code: `
				#!/bin/bash
				x=10
				y=$(calculate 20)
				z="hello"
				w=$(getResult)
			`,
			expected: map[string]string{
				"x": "10",
				"y": "$(calculate 20)",
				"z": "hello",
				"w": "$(getResult)",
			},
		},
		{
			name: "Makefile",
			code: `
				x := 10
				y := $(shell calculate 20)
				z := hello
				w := $(shell getResult)
			`,
			expected: map[string]string{
				"x": "10",
				"y": "$(shell calculate 20)",
				"z": "hello",
				"w": "$(shell getResult)",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var actual map[string]string

			if test.isGoCode {
				actual = TrackVariableAssignments(goCodeToAstFile(t, test.code))
			} else {
				actual = TrackVariableAssignments(shellOrMakefileCodeToAstFile(t, test.code))
			}

			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestTrackVariableAssignments_MultipleVariablesOnLeftHandSide(t *testing.T) {
	code := `
		package main

		func main() {
			x, y := 10, 20
			z := "hello"
			var w, v int
			w, v = 30, 40
		}
	`

	expected := map[string]string{
		"x": "10",
		"y": "20",
		"z": "hello",
		"w": "30",
		"v": "40",
	}

	actual := TrackVariableAssignments(goCodeToAstFile(t, code))
	assert.Equal(t, expected, actual)
}

func TestTrackVariableAssignments_DifferentFileTypes_TypeConversion(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		isGoCode bool
		expected map[string]string
	}{
		{
			name: "Go code",
			code: `
				package main

				func main() {
					x := 10
					y := float64(x)
					z := "hello"
					var w int
					w = int(y)
				}
			`,
			isGoCode: true,
			expected: map[string]string{
				"x": "10",
				"y": "float64(x)",
				"z": `"hello"`,
				"w": "int(y)",
			},
		},
		{
			name: "Makefile",
			code: `
				x := 10
				y := $(shell echo $(x) | awk '{print $$1 * 1.0}')
				z := hello
				w := $(shell echo $(y) | awk '{print int($$1)}')
			`,
			expected: map[string]string{
				"x": "10",
				"y": "$(shell echo $(x) | awk '{print $$1 * 1.0}')",
				"z": "hello",
				"w": "$(shell echo $(y) | awk '{print int($$1)}')",
			},
		},
		{
			name: "Shell script",
			code: `
				#!/bin/bash
				x=10
				y=$(echo $x | awk '{print $$1 * 1.0}')
				z="hello"
				w=$(echo $y | awk '{print int($$1)}')
			`,
			expected: map[string]string{
				"x": "10",
				"y": "$(echo $x | awk '{print $$1 * 1.0}')",
				"z": `"hello"`,
				"w": "$(echo $y | awk '{print int($$1)}')",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var actual map[string]string

			if test.isGoCode {
				actual = TrackVariableAssignments(goCodeToAstFile(t, test.code))
			} else {
				actual = TrackVariableAssignments(shellOrMakefileCodeToAstFile(t, test.code))
			}

			assert.Equal(t, test.expected, actual)
		})
	}
}

func goCodeToAstFile(t *testing.T, code string) *ast.File {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	require.NoError(t, err)

	return file
}

func shellOrMakefileCodeToAstFile(t *testing.T, code string) *ast.File {
	parser := syntax.NewParser()
	fileNode, err := parser.Parse(strToReader(code), "")
	require.NoError(t, err)

	return ConvertSyntaxNodeToAstFile(fileNode)
}

func strToReader(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}
