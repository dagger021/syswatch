package collector

import (
	"sort"
	"strings"
	"time"

	"github.com/dagger021/syswatch/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type Collector struct {
	lastNet  map[string]net.IOCountersStat
	lastTime time.Time
}

func New() *Collector {
	return &Collector{
		lastNet: make(map[string]net.IOCountersStat),
	}
}

func (c *Collector) Collect() (*model.Metrics, error) {
	totalCPU, _ := cpu.Percent(200*time.Millisecond, false)
	perCore, _ := cpu.Percent(0, true)

	m := &model.Metrics{UpdatedAt: time.Now(), Cores: perCore}

	if len(totalCPU) > 0 {
		m.CPU = totalCPU[0]
	}

	m.Memory.Virtual, _ = mem.VirtualMemory()
	m.Memory.Swap, _ = mem.SwapMemory()

	if partitions, err := disk.Partitions(false); err == nil {
		for _, p := range partitions {
			u, err := disk.Usage(p.Mountpoint)
			if err != nil || u.Total == 0 {
				continue
			}

			m.Disks = append(m.Disks, model.Disk{
				Mount:  p.Mountpoint,
				Device: p.Device,
				Used:   u.UsedPercent,
				Total:  u.Total,
			})
		}
	}

	c.collectNetwork(m)
	c.collectProcesses(m)

	return m, nil
}

func (c *Collector) collectNetwork(m *model.Metrics) {
	now := time.Now()
	elapsed := now.Sub(c.lastTime).Seconds()

	counters, err := net.IOCounters(true)
	if err != nil {
		return
	}

	next := make(map[string]net.IOCountersStat, len(counters))
	for _, n := range counters {
		next[n.Name] = n

		if strings.HasPrefix(n.Name, "lo") || elapsed <= 0 {
			continue
		}

		prev, ok := c.lastNet[n.Name]
		if !ok || n.BytesRecv < prev.BytesRecv || n.BytesSent < prev.BytesSent {
			continue
		}

		m.Networks = append(m.Networks, model.Network{
			Name: n.Name,
			Rx:   uint64(float64(n.BytesRecv-prev.BytesRecv) / elapsed),
			Tx:   uint64(float64(n.BytesSent-prev.BytesSent) / elapsed),
		})
	}

	c.lastNet = next
	c.lastTime = now
}

func (c *Collector) collectProcesses(m *model.Metrics) {
	allProcesses, err := process.Processes()
	if err != nil {
		return
	}

	for _, p := range allProcesses {
		name, err := p.Name()
		if err != nil {
			continue
		}

		cpuPercent, _ := p.CPUPercent()
		info, err := p.MemoryInfo()
		if err != nil || info == nil {
			continue
		}

		m.Processes = append(m.Processes, model.Process{
			PID:    p.Pid,
			Name:   name,
			CPU:    cpuPercent,
			Memory: info.RSS,
		})
	}

	sort.Slice(m.Processes, func(i, j int) bool {
		return m.Processes[i].CPU > m.Processes[j].CPU
	})

	// Top 10 processes by CPU usage
	if len(m.Processes) > 10 {
		m.Processes = m.Processes[:10]
	}
}
