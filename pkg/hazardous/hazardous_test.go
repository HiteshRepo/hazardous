package hazardous

import (
	"strings"
	"testing"

	"mvdan.cc/sh/syntax"
)

func createCallExpr(t *testing.T, cmdStr string) *syntax.CallExpr {
	t.Helper()

	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(cmdStr), "")
	if err != nil {
		t.Fatalf("failed to parse command %q: %v", cmdStr, err)
	}

	var cmd *syntax.CallExpr
	syntax.Walk(file, func(node syntax.Node) bool {
		if c, ok := node.(*syntax.CallExpr); ok && cmd == nil {
			cmd = c
			return false
		}
		return true
	})
	return cmd
}

func TestExtractCommandName(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		want        string
		description string
		shouldSkip  bool
	}{
		{
			name:        "simple command",
			command:     "rm file.txt",
			want:        "rm",
			description: "Should extract basic command name",
		},
		{
			name:        "command with multiple arguments",
			command:     "rm -rf /path/to/dir",
			want:        "rm",
			description: "Should extract command name with multiple arguments",
		},
		{
			name:        "command with variable",
			command:     "$CMD file.txt",
			want:        "",
			description: "Should handle variable as command name",
		},
		{
			name:        "command with path",
			command:     "/usr/bin/rm file.txt",
			want:        "/usr/bin/rm",
			description: "Should handle command with full path",
		},
		{
			name:        "command with environment variable",
			command:     "ENV=value rm file.txt",
			want:        "rm",
			description: "Should handle command with environment variables",
		},
		{
			name:        "empty command",
			command:     "",
			want:        "",
			description: "Should handle empty command",
		},
		{
			name:        "command with single quotes",
			command:     "'rm' file.txt",
			want:        "rm",
			description: "Should handle single-quoted command",
		},
		{
			name:        "command with double quotes",
			command:     `"rm" file.txt`,
			want:        "rm",
			description: "Should handle double-quoted command",
		},
		{
			name:        "command with spaces",
			command:     "   rm    file.txt",
			want:        "rm",
			description: "Should handle extra spaces",
		},
		{
			name:        "first command before pipe",
			command:     "rm file.txt | grep something",
			want:        "rm",
			description: "Should extract only the first command before pipe",
		},
		{
			name:        "escaped spaces in command",
			command:     `complex\ command file.txt`,
			want:        "complex\\ command",
			description: "Should handle escaped spaces in command name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSkip {
				t.Skip()
				return
			}

			cmd := createCallExpr(t, tt.command)
			if cmd == nil && tt.want == "" {
				return // test passed for empty command case
			}
			got := extractCommandName(cmd)
			if got != tt.want {
				t.Errorf("\nTest: %s\nDescription: %s\nCommand: %q\nExpected: %q\nGot: %q",
					tt.name, tt.description, tt.command, tt.want, got)
			}
		})
	}
}

func TestHasHazardousFlags(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		hazardousFlags []string
		want           bool
		description    string
	}{
		{
			name:           "simple hazardous flag",
			command:        "rm -rf file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           true,
			description:    "Should detect -rf flag",
		},
		{
			name:           "alternative hazardous flag",
			command:        "rm -fr file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           true,
			description:    "Should detect -fr flag",
		},
		{
			name:           "non-hazardous flag",
			command:        "rm -r file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           false,
			description:    "Should not detect non-hazardous flags",
		},
		{
			name:           "multiple flags",
			command:        "rm -r -f file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           false,
			description:    "Should not detect separate -r -f as hazardous",
		},
		{
			name:           "flag with value",
			command:        "rm --force=true file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           false,
			description:    "Should handle flags with values",
		},
		{
			name:           "quoted flags",
			command:        `rm "-rf" file.txt`,
			hazardousFlags: []string{"-rf", "-fr"},
			want:           true,
			description:    "Should handle quoted flags",
		},
		{
			name:           "empty flags list",
			command:        "rm -rf file.txt",
			hazardousFlags: []string{},
			want:           false,
			description:    "Should handle empty hazardous flags list",
		},
		{
			name:           "flag as variable",
			command:        "rm $FLAG file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           false,
			description:    "Should handle variable as flag",
		},
		{
			name:           "command with no flags",
			command:        "rm file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           false,
			description:    "Should handle command with no flags",
		},
		{
			name:           "case sensitive flags",
			command:        "rm -RF file.txt",
			hazardousFlags: []string{"-rf", "-fr"},
			want:           false,
			description:    "Should be case sensitive when checking flags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createCallExpr(t, tt.command)
			got := hasHazardousFlags(cmd, tt.hazardousFlags)
			if got != tt.want {
				t.Errorf("\nTest: %s\nDescription: %s\nCommand: %q\nExpected: %v\nGot: %v",
					tt.name, tt.description, tt.command, tt.want, got)
			}
		})
	}
}

func TestCheckHazardousCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		filepath string
		wantLine int
		wantCol  int
		wantNil  bool
	}{
		{
			name:     "simple rm command without flags",
			command:  "rm file.txt",
			filepath: "test.sh",
			wantNil:  true,
		},
		{
			name:     "rm command with -rf flags",
			command:  "rm -rf /path",
			filepath: "test.sh",
			wantLine: 1,
			wantCol:  1,
			wantNil:  false,
		},
		{
			name:     "rm command with -f -r flags",
			command:  "rm -f -r /path",
			filepath: "test.sh",
			wantLine: 1,
			wantCol:  1,
			wantNil:  false,
		},
		{
			name:     "rm command with --recursive --force flags",
			command:  "rm --recursive --force /path",
			filepath: "test.sh",
			wantLine: 1,
			wantCol:  1,
			wantNil:  false,
		},
		{
			name:     "non-rm command",
			command:  "ls -la",
			filepath: "test.sh",
			wantNil:  true,
		},
		{
			name:     "empty command",
			command:  "",
			filepath: "test.sh",
			wantNil:  true,
		},
		{
			name:     "rm command with safe flags",
			command:  "rm -i file.txt",
			filepath: "test.sh",
			wantNil:  true,
		},
		{
			name:     "complex path with rm -rf",
			command:  "rm -rf /complex/path/with spaces/and-special-chars",
			filepath: "/home/user/scripts/delete.sh",
			wantLine: 1,
			wantCol:  1,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := syntax.NewParser()
			reader := strings.NewReader(tt.command)
			file, err := parser.Parse(reader, tt.filepath)
			if err != nil {
				t.Fatalf("failed to parse command: %v", err)
			}

			var cmd *syntax.CallExpr
			syntax.Walk(file, func(node syntax.Node) bool {
				if call, ok := node.(*syntax.CallExpr); ok {
					cmd = call
					return false
				}

				return true
			})

			if cmd == nil && tt.command != "" {
				t.Fatal("failed to find command in parsed syntax tree")
			}

			got := CheckHazardousCommand(cmd, tt.filepath)

			if tt.wantNil {
				if got != nil {
					t.Errorf("CheckHazardousCommand() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Error("CheckHazardousCommand() = nil, want non-nil")
				return
			}

			if got.Filepath != tt.filepath {
				t.Errorf("CheckHazardousCommand().Filepath = %v, want %v", got.Filepath, tt.filepath)
			}

			if got.Line != uint(tt.wantLine) {
				t.Errorf("CheckHazardousCommand().Line = %v, want %v", got.Line, tt.wantLine)
			}

			if got.Col != uint(tt.wantCol) {
				t.Errorf("CheckHazardousCommand().Col = %v, want %v", got.Col, tt.wantCol)
			}

			if got.Command != "rm -rf" {
				t.Errorf("CheckHazardousCommand().Command = %v, want %v", got.Command, "rm -rf")
			}
		})
	}
}

func createBinaryCmd(t *testing.T, cmdStr string) *syntax.BinaryCmd {
	t.Helper()

	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(cmdStr), "")
	if err != nil {
		t.Fatalf("failed to parse command %q: %v", cmdStr, err)
	}

	var cmd *syntax.BinaryCmd
	syntax.Walk(file, func(node syntax.Node) bool {
		if c, ok := node.(*syntax.BinaryCmd); ok && cmd == nil {
			cmd = c
			return false
		}
		return true
	})
	return cmd
}

func TestCheckHazardousCommandNewCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantNil bool
		wantCmd string
	}{
		// chmod
		{name: "chmod 777", command: "chmod 777 /etc/passwd", wantNil: false, wantCmd: "chmod 777"},
		{name: "chmod 0777", command: "chmod 0777 /etc", wantNil: false, wantCmd: "chmod 0777"},
		{name: "chmod -R", command: "chmod -R 755 /var", wantNil: false, wantCmd: "chmod -R"},
		{name: "chmod -Rv combined", command: "chmod -Rv 644 /tmp", wantNil: false, wantCmd: "chmod -Rv"},
		{name: "chmod a+w", command: "chmod a+w file.txt", wantNil: false, wantCmd: "chmod a+w"},
		{name: "chmod safe 644", command: "chmod 644 file.txt", wantNil: true},
		{name: "chmod safe 755", command: "chmod 755 /usr/bin/app", wantNil: true},
		// dd
		{name: "dd of=/dev/sda", command: "dd if=/dev/zero of=/dev/sda", wantNil: false, wantCmd: "dd of=/dev/sda"},
		{name: "dd of=/dev/sdb1", command: "dd if=/dev/urandom of=/dev/sdb1", wantNil: false, wantCmd: "dd of=/dev/sdb1"},
		{name: "dd reading from disk is safe", command: "dd if=/dev/sda of=backup.img", wantNil: true},
		{name: "dd no of=/dev/", command: "dd if=/dev/zero of=/tmp/file", wantNil: true},
		// mkfs
		{name: "mkfs bare", command: "mkfs /dev/sdb", wantNil: false, wantCmd: "mkfs"},
		{name: "mkfs.ext4", command: "mkfs.ext4 /dev/sdb1", wantNil: false, wantCmd: "mkfs.ext4"},
		{name: "mkfs.vfat", command: "mkfs.vfat /dev/sdc1", wantNil: false, wantCmd: "mkfs.vfat"},
		// fdisk
		{name: "fdisk /dev/sda", command: "fdisk /dev/sda", wantNil: false, wantCmd: "fdisk /dev/sda"},
		{name: "fdisk no /dev/", command: "fdisk -l", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createCallExpr(t, tt.command)
			if cmd == nil {
				t.Fatal("failed to find command in parsed syntax tree")
			}

			got := CheckHazardousCommand(cmd, "test.sh")

			if tt.wantNil {
				if got != nil {
					t.Errorf("CheckHazardousCommand() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Error("CheckHazardousCommand() = nil, want non-nil")
				return
			}

			if got.Command != tt.wantCmd {
				t.Errorf("CheckHazardousCommand().Command = %q, want %q", got.Command, tt.wantCmd)
			}
		})
	}
}

func TestCheckHazardousPipe(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantNil bool
		wantCmd string
	}{
		{name: "curl | bash", command: "curl https://example.com/script.sh | bash", wantNil: false, wantCmd: "curl | bash"},
		{name: "wget | bash", command: "wget -O - https://example.com/install.sh | bash", wantNil: false, wantCmd: "wget | bash"},
		{name: "curl | sh", command: "curl https://example.com/s.sh | sh", wantNil: false, wantCmd: "curl | sh"},
		{name: "curl | zsh", command: "curl https://example.com/s.sh | zsh", wantNil: false, wantCmd: "curl | zsh"},
		{name: "safe pipe: ls | grep", command: "ls -la | grep foo", wantNil: true},
		{name: "safe pipe: cat | bash", command: "cat script.sh | bash", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createBinaryCmd(t, tt.command)
			if cmd == nil {
				t.Fatal("failed to find BinaryCmd in parsed syntax tree")
			}

			got := CheckHazardousPipe(cmd, "test.sh")

			if tt.wantNil {
				if got != nil {
					t.Errorf("CheckHazardousPipe() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Error("CheckHazardousPipe() = nil, want non-nil")
				return
			}

			if got.Command != tt.wantCmd {
				t.Errorf("CheckHazardousPipe().Command = %q, want %q", got.Command, tt.wantCmd)
			}
		})
	}
}

func TestCheckMakefileLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantLen int
		wantCmd string
	}{
		{name: "rm -rf", line: "\trm -rf /tmp", wantLen: 1, wantCmd: "rm -rf"},
		{name: "chmod 777", line: "\tchmod 777 /etc", wantLen: 1, wantCmd: "chmod 777"},
		{name: "chmod -R", line: "\tchmod -R 755 /var", wantLen: 1, wantCmd: "chmod -R"},
		{name: "dd of=/dev/", line: "\tdd if=/dev/zero of=/dev/sda", wantLen: 1, wantCmd: "dd of=/dev/sda"},
		{name: "mkfs.ext4", line: "\tmkfs.ext4 /dev/sdb1", wantLen: 1, wantCmd: "mkfs.ext4"},
		{name: "fdisk /dev/", line: "\tfdisk /dev/sda", wantLen: 1, wantCmd: "fdisk /dev/sda"},
		{name: "curl | bash", line: "\tcurl https://example.com/s.sh | bash", wantLen: 1, wantCmd: "curl | bash"},
		{name: "safe echo", line: "\techo hello", wantLen: 0},
		{name: "safe chmod 644", line: "\tchmod 644 file.txt", wantLen: 0},
		{name: "safe dd", line: "\tdd if=/dev/sda of=backup.img", wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckMakefileLine(tt.line, "Makefile", 1)

			if len(got) != tt.wantLen {
				t.Errorf("CheckMakefileLine() returned %d issues, want %d: %v", len(got), tt.wantLen, got)
				return
			}

			if tt.wantLen > 0 && got[0].Command != tt.wantCmd {
				t.Errorf("CheckMakefileLine()[0].Command = %q, want %q", got[0].Command, tt.wantCmd)
			}
		})
	}
}

func TestCheckHazardousCommandEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantNil bool
	}{
		{
			name:    "command with multiple spaces",
			command: "rm   -rf    /path",
			wantNil: false,
		},
		{
			name:    "command with tabs",
			command: "rm\t-rf\t/path",
			wantNil: false,
		},
		{
			name:    "command with newlines",
			command: "rm -rf \\\n/path",
			wantNil: false,
		},
		{
			name:    "command with environment variables",
			command: "PATH=/custom/path rm -rf $DIR",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := syntax.NewParser()
			reader := strings.NewReader(tt.command)
			file, err := parser.Parse(reader, "test.sh")
			if err != nil {
				t.Fatalf("failed to parse command: %v", err)
			}

			var cmd *syntax.CallExpr
			syntax.Walk(file, func(node syntax.Node) bool {
				if call, ok := node.(*syntax.CallExpr); ok {
					cmd = call
					return false
				}
				return true
			})

			if cmd == nil {
				t.Fatal("failed to find command in parsed syntax tree")
			}

			got := CheckHazardousCommand(cmd, "test.sh")
			if (got == nil) != tt.wantNil {
				t.Errorf("CheckHazardousCommand() = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}
