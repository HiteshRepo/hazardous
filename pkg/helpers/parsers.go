package helpers

import "strings"

// ExtractVarName extracts the variable name and remaining string from patterns like $(OUT_DIR) or ${OUT_DIR} or $VAR.
// It supports both $() and ${} formats, and also handles the simple $VAR format.
// If the input string does not match any of these formats, it returns an empty string for both the variable name and remaining string.
//
// Parameters:
// - arg: The input string containing the variable pattern.
//
// Returns:
// - A string representing the extracted variable name.
// - A string representing the remaining string after the extracted variable name.
func ExtractVarName(arg string) (string, string) {
	// Check for the $() format
	if strings.HasPrefix(arg, "$(") && strings.Contains(arg, ")") {
		start := strings.Index(arg, "$(") + 2
		end := strings.Index(arg, ")")
		return arg[start:end], arg[end+1:]
	}

	// Check for the ${} format
	if strings.HasPrefix(arg, "${") && strings.Contains(arg, "}") {
		start := strings.Index(arg, "${") + 2
		end := strings.Index(arg, "}")
		return arg[start:end], arg[end+1:]
	}

	// Check for the simple $VAR format
	if strings.HasPrefix(arg, "$") {
		varName := strings.TrimPrefix(arg, "$")
		if idx := strings.IndexAny(varName, "/ *"); idx != -1 {
			return varName[:idx], arg[idx+1:]
		}

		return varName, ""
	}

	return "", ""
}
