package model

import (
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

type Disk struct {
	Mount  string
	Device string
	Used   float64
	Total  uint64
}

type Memory struct {
	Virtual *mem.VirtualMemoryStat
	Swap    *mem.SwapMemoryStat
}

type Network struct {
	Name string
	Rx   uint64
	Tx   uint64
}

type Process struct {
	PID    int32
	Name   string
	CPU    float64
	Memory uint64
}

type Metrics struct {
	UpdatedAt time.Time
	CPU       float64
	Cores     []float64
	Memory    Memory
	Disks     []Disk
	Networks  []Network
	Processes []Process
}
