package ui

import "fmt"

var (
	memUnits = []string{"KB", "MB", "GB", "TB", "PB"}
)

func humanizeBytes(n uint64) string {
	const unit uint64 = 1024

	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := unit, 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}

	if exp >= len(memUnits) {
		exp = len(memUnits) - 1
	}

	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), memUnits[exp])
}

func humanizeRate(n uint64) string {
	return humanizeBytes(n) + "/s"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	// to avoid -- only "...".
	// e.g. - ("abc", 3) -- would have translated to "..."
	// or lower max -- to bounds out of range error
	if max <= 3 {
		return s[:max]
	}

	return s[:max-3] + "..."
}
