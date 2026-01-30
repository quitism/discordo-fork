package ui

import (
	"os"
	"strings"
)

// IsSixelSupported checks if the current terminal supports Sixel graphics.
// This is a heuristic based on environment variables.
func IsSixelSupported() bool {
	// Check explicitly enabled via env var (optional convention)
	if os.Getenv("ENABLE_SIXEL") == "1" {
		return true
	}

	term := strings.ToLower(os.Getenv("TERM"))
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))

	// Known terminals supporting Sixel
	if strings.Contains(term, "mlterm") ||
		strings.Contains(term, "foot") ||
		strings.Contains(term, "ghostty") {
		return true
	}

	// XTerm *might* support it, but it's not guaranteed.
	// We'll rely on the user to set ENABLE_SIXEL=1 or use a specific TERM if xterm.
    // Mac's iTerm2 supports inline images but via a proprietary protocol, not Sixel (unless using a bridge).
    // WezTerm supports Sixel.
    if strings.Contains(termProgram, "wezterm") {
        return true
    }

    // Mintty
    if strings.Contains(termProgram, "mintty") {
        return true
    }

	return false
}
