package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hiteshrepo/hazardous/pkg/issue"
	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanShellScript(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filepath string
		want     []issue.Issue
	}{
		{
			name: "single hazardous command",
			content: `#!/bin/bash
rm -rf /path/to/dir`,
			filepath: "test.sh",
			want: []issue.Issue{
				{
					Filepath: "test.sh",
					Line:     2,
					Col:      1,
					Command:  "rm -rf",
				},
			},
		},
		{
			name: "multiple hazardous commands",
			content: `#!/bin/bash
rm -rf /path1
echo "doing something"
rm -rf /path2`,
			filepath: "test.sh",
			want: []issue.Issue{
				{
					Filepath: "test.sh",
					Line:     2,
					Col:      1,
					Command:  "rm -rf",
				},
				{
					Filepath: "test.sh",
					Line:     4,
					Col:      1,
					Command:  "rm -rf",
				},
			},
		},
		{
			name: "no hazardous commands",
			content: `#!/bin/bash
echo "safe command"
ls -la
rm -i file`,
			filepath: "test.sh",
			want:     nil,
		},
		{
			name: "complex script with functions and conditions",
			content: `#!/bin/bash
function cleanup() {
    rm -rf /tmp/dir
}
if [ "$1" == "--clean" ]; then
    rm -rf ./build
fi`,
			filepath: "test.sh",
			want: []issue.Issue{
				{
					Filepath: "test.sh",
					Line:     3,
					Col:      5,
					Command:  "rm -rf",
				},
				{
					Filepath: "test.sh",
					Line:     6,
					Col:      5,
					Command:  "rm -rf",
				},
			},
		},
		{
			name: "script with syntax error",
			content: `#!/bin/bash
if [ "$1" == "--clean" ] then # missing semicolon
    rm -rf ./build
fi`,
			filepath: "test.sh",
			want:     nil,
		},
		{
			name:     "empty script",
			content:  "",
			filepath: "test.sh",
			want:     nil,
		},
		{
			name: "script with comments and whitespace",
			content: `#!/bin/bash
# This is a comment
   rm -rf /path  # Indented command with comment
	# More comments`,
			filepath: "test.sh",
			want: []issue.Issue{
				{
					Filepath: "test.sh",
					Line:     3,
					Col:      4,
					Command:  "rm -rf",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanShellScript(tt.content, tt.filepath)
			assert.Equal(t, got, tt.want, "scanShellScript() = %v, want %v", got, tt.want)
		})
	}
}

func TestScanMakefile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filepath string
		want     []issue.Issue
	}{
		{
			name: "simple rm -rf command",
			content: `all:
	rm -rf build/`,
			filepath: "Makefile",
			want: []issue.Issue{
				{
					Filepath: "Makefile",
					Line:     2,
					Col:      2,
					Command:  "rm -rf",
				},
			},
		},
		{
			name: "multiple rm commands",
			content: `clean:
	rm -rf build/
	rm -fr temp/
	rm -i src/`,
			filepath: "Makefile",
			want: []issue.Issue{
				{
					Filepath: "Makefile",
					Line:     2,
					Col:      2,
					Command:  "rm -rf",
				},
				{
					Filepath: "Makefile",
					Line:     3,
					Col:      2,
					Command:  "rm -fr",
				},
			},
		},
		{
			name: "rm commands with variables",
			content: `BUILD_DIR=build
clean:
	rm -rf $(BUILD_DIR)
	rm -rf ${BUILD_DIR}`,
			filepath: "Makefile",
			want: []issue.Issue{
				{
					Filepath: "Makefile",
					Line:     3,
					Col:      2,
					Command:  "rm -rf",
				},
				{
					Filepath: "Makefile",
					Line:     4,
					Col:      2,
					Command:  "rm -rf",
				},
			},
		},
		{
			name: "no hazardous commands",
			content: `all:
	echo "Building..."
	gcc -o prog main.c`,
			filepath: "Makefile",
			want:     nil,
		},
		{
			name: "commands with comments",
			content: `clean:
	# Remove build directory
	rm -rf build/ # Clean build
	# Remove temp files
	rm -fr temp/ # Clean temp`,
			filepath: "Makefile",
			want: []issue.Issue{
				{
					Filepath: "Makefile",
					Line:     3,
					Col:      2,
					Command:  "rm -rf",
				},
				{
					Filepath: "Makefile",
					Line:     5,
					Col:      2,
					Command:  "rm -fr",
				},
			},
		},
		{
			name:     "empty Makefile",
			content:  "",
			filepath: "Makefile",
			want:     nil,
		},
		{
			name: "indented and spaced variations",
			content: `clean:
	  rm -rf build/
		rm  -rf  temp/
	rm	-rf	data/`,
			filepath: "Makefile",
			want: []issue.Issue{
				{
					Filepath: "Makefile",
					Line:     2,
					Col:      4,
					Command:  "rm -rf",
				},
				{
					Filepath: "Makefile",
					Line:     3,
					Col:      3,
					Command:  "rm -rf",
				},
				{
					Filepath: "Makefile",
					Line:     4,
					Col:      2,
					Command:  "rm -rf",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanMakefile(tt.content, tt.filepath)
			assert.Equal(t, tt.want, got, "scanMakefile() = %v, want %v", got, tt.want)
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"hazardous": func() int {
			main()

			return 0
		},
	}))
}

func TestScripts(t *testing.T) {
	t.Parallel()

	var goEnv struct {
		GOCACHE    string
		GOMODCACHE string
		GOMOD      string
	}
	out, err := exec.Command("go", "env", "-json").CombinedOutput()
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(out, &goEnv))

	p := testscript.Params{
		Dir: filepath.Join("testdata", "scripts"),
		Setup: func(env *testscript.Env) error {
			env.Setenv("GOCACHE", goEnv.GOCACHE)
			env.Setenv("GOMODCACHE", goEnv.GOMODCACHE)
			env.Setenv("GOMOD_DIR", filepath.Dir(goEnv.GOMOD))
			return nil
		},
	}
	require.NoError(t, gotooltest.Setup(&p))

	testscript.Run(t, p)
}
