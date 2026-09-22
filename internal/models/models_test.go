package models

import (
	"testing"
)

func TestTableNames(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Team", Team{}.TableName(), "teams"},
		{"User", User{}.TableName(), "users"},
		{"TaskStatus", TaskStatus{}.TableName(), "task_statuses"},
		{"TaskAction", TaskAction{}.TableName(), "task_actions"},
		{"Task", Task{}.TableName(), "tasks"},
		{"TaskLog", TaskLog{}.TableName(), "task_logs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected table name %q, got %q", tt.expected, tt.got)
			}
		})
	}
}

func TestJSONMap_ValueAndScan(t *testing.T) {
	m := JSONMap{"key": "value", "count": float64(42)}

	// Test Value
	val, err := m.Value()
	if err != nil {
		t.Fatalf("unexpected error on Value(): %v", err)
	}

	// Test Scan with []byte
	var scanned JSONMap
	if err := scanned.Scan(val); err != nil {
		t.Fatalf("unexpected error on Scan([]byte): %v", err)
	}
	if scanned["key"] != "value" || scanned["count"] != float64(42) {
		t.Fatalf("scanned data mismatch: %+v", scanned)
	}

	// Test Scan with string
	var scannedStr JSONMap
	if err := scannedStr.Scan(`{"foo":"bar"}`); err != nil {
		t.Fatalf("unexpected error on Scan(string): %v", err)
	}
	if scannedStr["foo"] != "bar" {
		t.Fatalf("scanned data mismatch for string: %+v", scannedStr)
	}

	// Test Scan with nil
	var scannedNil JSONMap
	if err := scannedNil.Scan(nil); err != nil {
		t.Fatalf("unexpected error on Scan(nil): %v", err)
	}
	if len(scannedNil) != 0 {
		t.Fatalf("expected empty map for Scan(nil), got %+v", scannedNil)
	}

	// Test Scan with empty bytes
	var scannedEmpty JSONMap
	if err := scannedEmpty.Scan([]byte{}); err != nil {
		t.Fatalf("unexpected error on Scan([]byte{}): %v", err)
	}
	if len(scannedEmpty) != 0 {
		t.Fatalf("expected empty map for Scan([]byte{}), got %+v", scannedEmpty)
	}

	// Test Value for nil map
	var nilMap JSONMap
	nilVal, err := nilMap.Value()
	if err != nil {
		t.Fatalf("unexpected error on nil Value(): %v", err)
	}
	if nilVal != "{}" {
		t.Fatalf("expected '{}' for nil map Value(), got %v", nilVal)
	}

	// Test Scan with invalid type
	var invalidMap JSONMap
	if err := invalidMap.Scan(12345); err == nil {
		t.Fatal("expected error on Scan with unsupported type, got nil")
	}
}
