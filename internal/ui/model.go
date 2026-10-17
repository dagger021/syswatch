package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/dagger021/syswatch/internal/collector"
	cfg "github.com/dagger021/syswatch/internal/config"
	"github.com/dagger021/syswatch/internal/model"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type tickMsg time.Time

type metricsMsg struct {
	data *model.Metrics
	err  error
}

type UI struct {
	collector *collector.Collector
	metrics   *model.Metrics
	err       error
	cursor    int
	sortBy    string
}

func New(c *collector.Collector) UI {
	return UI{collector: c, sortBy: "cpu"}
}

func (u UI) Init() tea.Cmd {
	return tea.Batch(collectCmd(u.collector), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func collectCmd(c *collector.Collector) tea.Cmd {
	return func() tea.Msg {
		data, err := c.Collect()
		return metricsMsg{data: data, err: err}
	}
}

func (u UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case metricsMsg:
		if msg.err != nil {
			u.err = msg.err
		} else {
			u.metrics, u.err = msg.data, nil
		}
		return u, tickCmd()

	case tickMsg:
		return u, collectCmd(u.collector)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return u, tea.Quit

		case "r":
			return u, collectCmd(u.collector)

		case "up", "k":
			if u.cursor > 0 {
				u.cursor--
			}

		case "down", "j":
			if u.metrics != nil && u.cursor < len(u.metrics.Processes)-1 {
				u.cursor++
			}

		case "c":
			u.sortBy = "cpu"

		case "m":
			u.sortBy = "memory"
		}
	}

	return u, nil
}

func (u UI) View() tea.View {
	var b strings.Builder

	b.WriteString(titleStyle.Render(cfg.TITLE))
	b.WriteString("\n")

	if u.err != nil {
		fmt.Fprintf(&b, "Collector error: %s\n", u.err.Error())
	}

	if u.metrics == nil {
		b.WriteString("\nCollecting system resource metrics...\n")
	} else {
		x := u.metrics

		b.WriteString(mutedStyle.Render(
			fmt.Sprintf("Updated: %s", x.UpdatedAt.Format("15:04:05")),
		))

		b.WriteString("\n\n")
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  Total: %5.1f%%", x.CPU)))
		b.WriteString("\n")

		b.WriteString(renderCores(u.metrics.Cores, 7))

		b.WriteString("\n\n")
		b.WriteString(sectionStyle.Render("MEMORY"))
		b.WriteString("\n")

		if mem := x.Memory.Virtual; mem != nil {
			fmt.Fprintf(
				&b,
				"  RAM  %5.1f%%  %s / %s\n",
				mem.UsedPercent, humanizeBytes(mem.Used), humanizeBytes(mem.Total),
			)
		}

		if mem := x.Memory.Swap; mem != nil {
			fmt.Fprintf(
				&b,
				"  Swap %5.1f%%  %s / %s\n",
				mem.UsedPercent, humanizeBytes(mem.Used), humanizeBytes(mem.Total),
			)
		}

		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("DISKS"))
		b.WriteString("\n")

		for _, d := range u.metrics.Disks {
			fmt.Fprintf(
				&b,
				"  %-18s %5.1f%%  %s\n",
				truncate(d.Mount, 18), d.Used, humanizeBytes(d.Total),
			)
		}

		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("NETWORK"))
		b.WriteString("\n")

		if len(x.Networks) == 0 {
			b.WriteString("  Measuring network throughput...\n")
		}

		for _, n := range x.Networks {
			fmt.Fprintf(
				&b,
				"  %-12s  RX %-10s  TX %-10s\n",
				truncate(n.Name, 12), humanizeRate(n.Rx), humanizeRate(n.Tx),
			)
		}

		// copying processes to avoid mutating collector state
		procs := append([]model.Process(nil), x.Processes...)
		if u.sortBy == "memory" {
			slices.SortStableFunc(procs, func(a, b model.Process) int {
				return int(b.Memory) - int(a.Memory)
			})
		} else {
			slices.SortStableFunc(procs, func(a, b model.Process) int {
				return int(b.CPU) - int(a.CPU)
			})

		}

		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("PROCESSES"))

		b.WriteString("\n")
		b.WriteString("  PID      CPU(%)    RSS        NAME\n")

		for i, p := range procs {
			line := fmt.Sprintf(
				"  %-7d  %6.1f  %-9s  %s",
				p.PID, p.CPU, humanizeBytes(p.Memory), truncate(p.Name, 30),
			)

			if i == u.cursor {
				line = selectedStyle.Render("> " + strings.TrimLeft(line, " "))
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(
		"↑/↓ navigate c CPU sort | m memory sort | r refresh | q quit",
	))

	v := tea.NewView(b.String())
	v.AltScreen = true

	return v
}

func renderCores(cores []float64, columns int) string {
	if columns < 1 {
		columns = len(cores) / 4
	}

	var b strings.Builder

	for i, usage := range cores {
		if i%columns == 0 {
			if i > 0 {
				b.WriteString("\n")
			}
		} else {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "  C%-2d %5.1f%%", i, usage)
	}

	return b.String()
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7DD3FC"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A7F3D0"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FDE68A"))
)
