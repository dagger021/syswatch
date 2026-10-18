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
	var out, top, left, right strings.Builder

	top.WriteString(titleStyle.Render(cfg.TITLE))
	top.WriteString("\n")

	if u.err != nil {
		fmt.Fprintf(&top, "Collector error: %s\n", u.err.Error())
	}

	if u.metrics == nil {
		top.WriteString("\nCollecting system resource metrics...\n")
	} else {
		x := u.metrics

		top.WriteString(mutedStyle.Render(
			fmt.Sprintf("Updated: %s", x.UpdatedAt.Format("15:04:05")),
		))

		top.WriteString("\n")
		renderSection(&top, "CPU CORES")
		top.WriteString(mutedStyle.Render(fmt.Sprintf("  Total: %5.1f%%", x.CPU)))
		top.WriteString("\n")

		renderCores(&top, u.metrics.Cores, 7)

		renderSection(&left, "MEMORY")
		renderMemory(&left, x.Memory)

		renderSection(&right, "DISKS")
		renderDisks(&right, u.metrics.Disks)

		renderSection(&right, "NETWORKS")
		renderNetworks(&right, x.Networks)

		// copying processes to avoid mutating collector state
		processes := append([]model.Process(nil), x.Processes...)
		if u.sortBy == "memory" {
			slices.SortStableFunc(processes, func(a, b model.Process) int {
				return int(b.Memory) - int(a.Memory)
			})
		} else {
			slices.SortStableFunc(processes, func(a, b model.Process) int {
				return int(b.CPU) - int(a.CPU)
			})

		}

		renderSection(&left, "PROCESSES")
		renderProcesses(&left, processes, u.cursor)
	}

	out.WriteString(lipgloss.JoinVertical(
		lipgloss.Left,
		top.String(),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, left.String(), "  ", right.String()),
	))

	out.WriteString("\n")
	out.WriteString(mutedStyle.Render(
		"↑/↓ navigate c CPU sort | m memory sort | r refresh | q quit",
	))

	v := tea.NewView(out.String())
	v.AltScreen = true

	return v
}

func renderSection(b *strings.Builder, sectionTitle string) {
	fmt.Fprintf(b, "\n%s\n", sectionStyle.Render(sectionTitle))
}

func renderCores(b *strings.Builder, cores []float64, columns int) {
	if columns < 1 {
		columns = len(cores) / 4
	}

	for i, usage := range cores {
		if i%columns == 0 {
			if i > 0 {
				b.WriteString("\n")
			}
		} else {
			b.WriteString("  ")
		}
		fmt.Fprintf(b, "  C%-2d %5.1f%%", i, usage)
	}

	// b.WriteString("\n")
}

func renderMemory(b *strings.Builder, memory model.Memory) {
	if mem := memory.Virtual; mem != nil {
		fmt.Fprintf(
			b,
			"  RAM  %5.1f%%  %s / %s\n",
			mem.UsedPercent, humanizeBytes(mem.Used), humanizeBytes(mem.Total),
		)
	}

	if mem := memory.Swap; mem != nil {
		fmt.Fprintf(
			b,
			"  Swap %5.1f%%  %s / %s\n",
			mem.UsedPercent, humanizeBytes(mem.Used), humanizeBytes(mem.Total),
		)
	}

}

func renderDisks(b *strings.Builder, disks []model.Disk) {
	for _, d := range disks {
		fmt.Fprintf(
			b,
			"  %-18s %5.1f%%  %s\n",
			truncate(d.Mount, 18), d.Used, humanizeBytes(d.Total),
		)
	}
}

func renderNetworks(b *strings.Builder, networks []model.Network) {
	if len(networks) == 0 {
		b.WriteString("  Measuring network throughput...\n")
	}

	for _, n := range networks {
		fmt.Fprintf(
			b,
			"  %-20s  RX %-10s  TX %-10s\n",
			truncate(n.Name, 20), humanizeRate(n.Rx), humanizeRate(n.Tx),
		)
	}
}

func renderProcesses(b *strings.Builder, processes []model.Process, cursor int) {
	b.WriteString("  PID      CPU(%)    RSS        NAME\n")

	for i, p := range processes {
		line := fmt.Sprintf(
			"  %-7d  %6.1f  %-9s  %s",
			p.PID, p.CPU, humanizeBytes(p.Memory), truncate(p.Name, 30),
		)

		if i == cursor {
			line = selectedStyle.Render("> " + strings.TrimLeft(line, " "))
		}

		b.WriteString(line)
		b.WriteString("\n")
	}
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
