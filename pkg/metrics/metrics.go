package metrics

import (
	"fmt"
	"github.com/Eressleep/metrics-tpl/internal/model"
	"math/rand"
	"runtime"
	"sync"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Metrics struct {
	// Runtime метрики
	Alloc         float64
	BuckHashSys   float64
	Frees         float64
	GCCPUFraction float64
	GCSys         float64
	HeapAlloc     float64
	HeapIdle      float64
	HeapInuse     float64
	HeapObjects   float64
	HeapReleased  float64
	HeapSys       float64
	LastGC        float64
	Lookups       float64
	MCacheInuse   float64
	MCacheSys     float64
	MSpanInuse    float64
	MSpanSys      float64
	Mallocs       float64
	NextGC        float64
	NumForcedGC   float64
	NumGC         float64
	OtherSys      float64
	PauseTotalNs  float64
	StackInuse    float64
	StackSys      float64
	Sys           float64
	TotalAlloc    float64

	TotalMemory     float64   `json:"TotalMemory"`
	FreeMemory      float64   `json:"FreeMemory"`
	CPUutilization1 []float64 `json:"CPUutilization1"`

	PollCount   int64   // counter
	RandomValue float64 // gauge

	mu sync.RWMutex
}

func New() *Metrics {
	return &Metrics{
		CPUutilization1: make([]float64, 0),
	}
}

func (m *Metrics) UpdateRuntime() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Alloc = float64(memStats.Alloc)
	m.BuckHashSys = float64(memStats.BuckHashSys)
	m.Frees = float64(memStats.Frees)
	m.GCCPUFraction = memStats.GCCPUFraction
	m.GCSys = float64(memStats.GCSys)
	m.HeapAlloc = float64(memStats.HeapAlloc)
	m.HeapIdle = float64(memStats.HeapIdle)
	m.HeapInuse = float64(memStats.HeapInuse)
	m.HeapObjects = float64(memStats.HeapObjects)
	m.HeapReleased = float64(memStats.HeapReleased)
	m.HeapSys = float64(memStats.HeapSys)
	m.LastGC = float64(memStats.LastGC)
	m.Lookups = float64(memStats.Lookups)
	m.MCacheInuse = float64(memStats.MCacheInuse)
	m.MCacheSys = float64(memStats.MCacheSys)
	m.MSpanInuse = float64(memStats.MSpanInuse)
	m.MSpanSys = float64(memStats.MSpanSys)
	m.Mallocs = float64(memStats.Mallocs)
	m.NextGC = float64(memStats.NextGC)
	m.NumForcedGC = float64(memStats.NumForcedGC)
	m.NumGC = float64(memStats.NumGC)
	m.OtherSys = float64(memStats.OtherSys)
	m.PauseTotalNs = float64(memStats.PauseTotalNs)
	m.StackInuse = float64(memStats.StackInuse)
	m.StackSys = float64(memStats.StackSys)
	m.Sys = float64(memStats.Sys)
	m.TotalAlloc = float64(memStats.TotalAlloc)
}

func (m *Metrics) UpdateGopsutil() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if vmem, err := mem.VirtualMemory(); err == nil {
		m.TotalMemory = float64(vmem.Total)
		m.FreeMemory = float64(vmem.Free)
	}

	if cpuPercent, err := cpu.Percent(0, true); err == nil {
		m.CPUutilization1 = cpuPercent
	}
}

func (m *Metrics) UpdateRandom() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RandomValue = rand.Float64()
}

func (m *Metrics) IncrementPollCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PollCount++
}

func (m *Metrics) GetAllGauges() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gauges := map[string]float64{
		"Alloc":         m.Alloc,
		"BuckHashSys":   m.BuckHashSys,
		"Frees":         m.Frees,
		"GCCPUFraction": m.GCCPUFraction,
		"GCSys":         m.GCSys,
		"HeapAlloc":     m.HeapAlloc,
		"HeapIdle":      m.HeapIdle,
		"HeapInuse":     m.HeapInuse,
		"HeapObjects":   m.HeapObjects,
		"HeapReleased":  m.HeapReleased,
		"HeapSys":       m.HeapSys,
		"LastGC":        m.LastGC,
		"Lookups":       m.Lookups,
		"MCacheInuse":   m.MCacheInuse,
		"MCacheSys":     m.MCacheSys,
		"MSpanInuse":    m.MSpanInuse,
		"MSpanSys":      m.MSpanSys,
		"Mallocs":       m.Mallocs,
		"NextGC":        m.NextGC,
		"NumForcedGC":   m.NumForcedGC,
		"NumGC":         m.NumGC,
		"OtherSys":      m.OtherSys,
		"PauseTotalNs":  m.PauseTotalNs,
		"StackInuse":    m.StackInuse,
		"StackSys":      m.StackSys,
		"Sys":           m.Sys,
		"TotalAlloc":    m.TotalAlloc,
		"RandomValue":   m.RandomValue,
		"TotalMemory":   m.TotalMemory,
		"FreeMemory":    m.FreeMemory,
	}

	for i, cpuPercent := range m.CPUutilization1 {
		gauges[fmt.Sprintf("CPUutilization%d", i+1)] = cpuPercent
	}

	return gauges
}

func (m *Metrics) ToMetricsSlice() []model.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []model.Metrics

	gauges := map[string]float64{
		"Alloc":         m.Alloc,
		"BuckHashSys":   m.BuckHashSys,
		"Frees":         m.Frees,
		"GCCPUFraction": m.GCCPUFraction,
		"GCSys":         m.GCSys,
		"HeapAlloc":     m.HeapAlloc,
		"HeapIdle":      m.HeapIdle,
		"HeapInuse":     m.HeapInuse,
		"HeapObjects":   m.HeapObjects,
		"HeapReleased":  m.HeapReleased,
		"HeapSys":       m.HeapSys,
		"LastGC":        m.LastGC,
		"Lookups":       m.Lookups,
		"MCacheInuse":   m.MCacheInuse,
		"MCacheSys":     m.MCacheSys,
		"MSpanInuse":    m.MSpanInuse,
		"MSpanSys":      m.MSpanSys,
		"Mallocs":       m.Mallocs,
		"NextGC":        m.NextGC,
		"NumForcedGC":   m.NumForcedGC,
		"NumGC":         m.NumGC,
		"OtherSys":      m.OtherSys,
		"PauseTotalNs":  m.PauseTotalNs,
		"StackInuse":    m.StackInuse,
		"StackSys":      m.StackSys,
		"Sys":           m.Sys,
		"TotalAlloc":    m.TotalAlloc,
		"RandomValue":   m.RandomValue,
		"TotalMemory":   m.TotalMemory,
		"FreeMemory":    m.FreeMemory,
	}

	for name, val := range gauges {
		v := val
		result = append(result, model.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}

	for i, cpuPercent := range m.CPUutilization1 {
		v := cpuPercent
		result = append(result, model.Metrics{
			ID:    fmt.Sprintf("CPUutilization%d", i+1),
			MType: "gauge",
			Value: &v,
		})
	}

	pollCount := m.PollCount
	result = append(result, model.Metrics{
		ID:    "PollCount",
		MType: "counter",
		Delta: &pollCount,
	})

	return result
}

func (m *Metrics) GetPollCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.PollCount
}

func (m *Metrics) Copy() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cpuCopy := make([]float64, len(m.CPUutilization1))
	copy(cpuCopy, m.CPUutilization1)

	return &Metrics{
		Alloc:           m.Alloc,
		BuckHashSys:     m.BuckHashSys,
		Frees:           m.Frees,
		GCCPUFraction:   m.GCCPUFraction,
		GCSys:           m.GCSys,
		HeapAlloc:       m.HeapAlloc,
		HeapIdle:        m.HeapIdle,
		HeapInuse:       m.HeapInuse,
		HeapObjects:     m.HeapObjects,
		HeapReleased:    m.HeapReleased,
		HeapSys:         m.HeapSys,
		LastGC:          m.LastGC,
		Lookups:         m.Lookups,
		MCacheInuse:     m.MCacheInuse,
		MCacheSys:       m.MCacheSys,
		MSpanInuse:      m.MSpanInuse,
		MSpanSys:        m.MSpanSys,
		Mallocs:         m.Mallocs,
		NextGC:          m.NextGC,
		NumForcedGC:     m.NumForcedGC,
		NumGC:           m.NumGC,
		OtherSys:        m.OtherSys,
		PauseTotalNs:    m.PauseTotalNs,
		StackInuse:      m.StackInuse,
		StackSys:        m.StackSys,
		Sys:             m.Sys,
		TotalAlloc:      m.TotalAlloc,
		TotalMemory:     m.TotalMemory,
		FreeMemory:      m.FreeMemory,
		CPUutilization1: cpuCopy,
		PollCount:       m.PollCount,
		RandomValue:     m.RandomValue,
	}
}
