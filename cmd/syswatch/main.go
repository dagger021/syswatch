package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/dagger021/syswatch/internal/collector"
	"github.com/dagger021/syswatch/internal/ui"
)

func main() {
	c := collector.New()
	app := tea.NewProgram(ui.New(c))

	if _, err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "syswatch:", err)
		os.Exit(1)
	}
}
