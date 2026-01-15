package storage

import (
	"sync"
)

// MemStorage хранилище метрик в памяти
type MemStorage struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]float64
}

// NewMemStorage создает новое хранилище в памяти
func NewMemStorage() *MemStorage {
	return &MemStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

// UpdateCounter обновляет значение counter метрики (добавляет к существующему)
func (s *MemStorage) UpdateCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

// UpdateGauge обновляет значение gauge метрики (заменяет существующее)
func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

// GetCounter возвращает значение counter метрики
func (s *MemStorage) GetCounter(name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.counters[name]
	return value, ok
}

// GetGauge возвращает значение gauge метрики
func (s *MemStorage) GetGauge(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.gauges[name]
	return value, ok
}

// GetAllCounters возвращает все counter метрики
func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]int64)
	for k, v := range s.counters {
		result[k] = v
	}
	return result
}

// GetAllGauges возвращает все gauge метрики
func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]float64)
	for k, v := range s.gauges {
		result[k] = v
	}
	return result
}
