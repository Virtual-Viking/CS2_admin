package steam

import (
	"regexp"
	"strconv"
)

// Progress represents SteamCMD operation progress.
type Progress struct {
	Stage   string  // "downloading", "validating", "installing", "complete", "error"
	Percent float64 // 0-100
	Message string  // human-readable message
}

var (
	// Update state (0x61) downloading, progress: 45.23 (12345 / 67890)
	progressRe = regexp.MustCompile(`Update state \(0x[0-9a-fA-F]+\) (\w+), progress:\s*([\d.]+)`)
	// Success! App '730' fully installed.
	successRe = regexp.MustCompile(`Success! App '\d+' fully installed\.`)
	// Error!
	errorRe = regexp.MustCompile(`^Error!`)
)

// ParseProgressLine parses a SteamCMD stdout line and returns a Progress struct if the line
// contains progress information. Returns nil for non-relevant lines.
func ParseProgressLine(line string) *Progress {
	if line == "" {
		return nil
	}

	// Error
	if errorRe.MatchString(line) {
		return &Progress{Stage: "error", Percent: 0, Message: line}
	}

	// Success/complete
	if successRe.MatchString(line) {
		return &Progress{Stage: "complete", Percent: 100, Message: "Installation complete"}
	}

	// Update state: downloading, validating, preallocating, reconfiguring, etc.
	m := progressRe.FindStringSubmatch(line)
	if m != nil {
		stage := m[1]
		percent, _ := strconv.ParseFloat(m[2], 64)

		// Normalize stage names
		switch stage {
		case "downloading":
		case "validating":
		case "preallocating", "reconfiguring":
			stage = "installing"
		}

		return &Progress{
			Stage:   stage,
			Percent: percent,
			Message: line,
		}
	}

	return nil
}
