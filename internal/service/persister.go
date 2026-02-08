package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Asfode1/go-musthave-metrics/internal/model"
	"github.com/Asfode1/go-musthave-metrics/internal/storage"
)

// Persister сохраняет/восстанавливает метрики на диск.
type Persister struct {
	storage *storage.MemStorage
	path    string
}

func NewPersister(s *storage.MemStorage, path string) *Persister {
	return &Persister{storage: s, path: path}
}

// Load загружает метрики из файла, если файл существует.
func (p *Persister) Load() error {
	if p.path == "" {
		return nil
	}
	b, err := os.ReadFile(p.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read file: %w", err)
	}
	if len(b) == 0 {
		return nil
	}

	var metrics []model.Metrics
	if err := json.Unmarshal(b, &metrics); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	for _, m := range metrics {
		if m.ID == "" {
			continue
		}
		switch m.MType {
		case model.Counter:
			if m.Delta == nil {
				continue
			}
			p.storage.SetCounter(m.ID, *m.Delta)
		case model.Gauge:
			if m.Value == nil {
				continue
			}
			p.storage.UpdateGauge(m.ID, *m.Value)
		}
	}

	return nil
}

// Save сохраняет текущие метрики в файл.
func (p *Persister) Save() error {
	if p.path == "" {
		return nil
	}

	metrics := p.snapshot()
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	dir := filepath.Dir(p.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}

	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}

	// Windows: os.Rename не заменяет существующий файл — удаляем старый
	_ = os.Remove(p.path)
	if err := os.Rename(tmp, p.path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// StartPeriodic запускает периодическое сохранение. Если interval <= 0 — ничего не запускает.
func (p *Persister) StartPeriodic(interval time.Duration, onError func(error)) {
	if interval <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			if err := p.Save(); err != nil && onError != nil {
				onError(err)
			}
		}
	}()
}

func (p *Persister) snapshot() []model.Metrics {
	counters := p.storage.GetAllCounters()
	gauges := p.storage.GetAllGauges()

	out := make([]model.Metrics, 0, len(counters)+len(gauges))
	for id, v := range counters {
		val := v
		out = append(out, model.Metrics{
			ID:    id,
			MType: model.Counter,
			Delta: &val,
		})
	}
	for id, v := range gauges {
		val := v
		out = append(out, model.Metrics{
			ID:    id,
			MType: model.Gauge,
			Value: &val,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

