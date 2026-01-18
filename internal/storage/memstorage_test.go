package storage

import (
	"testing"
)

func TestMemStorage_UpdateCounter(t *testing.T) {
	storage := NewMemStorage()

	// Первое обновление
	storage.UpdateCounter("testCounter", 10)
	value, ok := storage.GetCounter("testCounter")
	if !ok {
		t.Fatal("Counter should exist")
	}
	if value != 10 {
		t.Errorf("Expected 10, got %d", value)
	}

	// Второе обновление (должно добавиться)
	storage.UpdateCounter("testCounter", 5)
	value, ok = storage.GetCounter("testCounter")
	if !ok {
		t.Fatal("Counter should exist")
	}
	if value != 15 {
		t.Errorf("Expected 15, got %d", value)
	}
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	storage := NewMemStorage()

	// Первое обновление
	storage.UpdateGauge("testGauge", 3.14)
	value, ok := storage.GetGauge("testGauge")
	if !ok {
		t.Fatal("Gauge should exist")
	}
	if value != 3.14 {
		t.Errorf("Expected 3.14, got %f", value)
	}

	// Второе обновление (должно замениться)
	storage.UpdateGauge("testGauge", 2.71)
	value, ok = storage.GetGauge("testGauge")
	if !ok {
		t.Fatal("Gauge should exist")
	}
	if value != 2.71 {
		t.Errorf("Expected 2.71, got %f", value)
	}
}

func TestMemStorage_GetCounter_NotExists(t *testing.T) {
	storage := NewMemStorage()
	_, ok := storage.GetCounter("nonExistent")
	if ok {
		t.Error("Counter should not exist")
	}
}

func TestMemStorage_GetGauge_NotExists(t *testing.T) {
	storage := NewMemStorage()
	_, ok := storage.GetGauge("nonExistent")
	if ok {
		t.Error("Gauge should not exist")
	}
}

func TestMemStorage_GetAllCounters(t *testing.T) {
	storage := NewMemStorage()
	storage.UpdateCounter("counter1", 10)
	storage.UpdateCounter("counter2", 20)

	counters := storage.GetAllCounters()
	if len(counters) != 2 {
		t.Errorf("Expected 2 counters, got %d", len(counters))
	}
	if counters["counter1"] != 10 {
		t.Errorf("Expected counter1 to be 10, got %d", counters["counter1"])
	}
	if counters["counter2"] != 20 {
		t.Errorf("Expected counter2 to be 20, got %d", counters["counter2"])
	}
}

func TestMemStorage_GetAllGauges(t *testing.T) {
	storage := NewMemStorage()
	storage.UpdateGauge("gauge1", 1.1)
	storage.UpdateGauge("gauge2", 2.2)

	gauges := storage.GetAllGauges()
	if len(gauges) != 2 {
		t.Errorf("Expected 2 gauges, got %d", len(gauges))
	}
	if gauges["gauge1"] != 1.1 {
		t.Errorf("Expected gauge1 to be 1.1, got %f", gauges["gauge1"])
	}
	if gauges["gauge2"] != 2.2 {
		t.Errorf("Expected gauge2 to be 2.2, got %f", gauges["gauge2"])
	}
}
