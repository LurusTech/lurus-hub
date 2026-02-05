package pool

import (
	"bytes"
	"testing"
)

func TestGetBuffer(t *testing.T) {
	buf := GetBuffer()
	if buf == nil {
		t.Fatal("GetBuffer returned nil")
	}
	buf.WriteString("test")
	if buf.String() != "test" {
		t.Errorf("expected 'test', got %q", buf.String())
	}
	PutBuffer(buf)
}

func TestGetIntSlice(t *testing.T) {
	slice := GetIntSlice()
	if slice == nil {
		t.Fatal("GetIntSlice returned nil")
	}
	*slice = append(*slice, 1, 2, 3)
	if len(*slice) != 3 {
		t.Errorf("expected length 3, got %d", len(*slice))
	}
	PutIntSlice(slice)
}

func TestGetStringBuilder(t *testing.T) {
	sb := GetStringBuilder()
	if sb == nil {
		t.Fatal("GetStringBuilder returned nil")
	}
	sb.WriteString("hello")
	if sb.String() != "hello" {
		t.Errorf("expected 'hello', got %q", sb.String())
	}
	PutStringBuilder(sb)
}

func TestGetIntBoolMap(t *testing.T) {
	m := GetIntBoolMap()
	if m == nil {
		t.Fatal("GetIntBoolMap returned nil")
	}
	m[1] = true
	m[2] = false
	if len(m) != 2 {
		t.Errorf("expected length 2, got %d", len(m))
	}
	PutIntBoolMap(m)

	// Get again and verify it's cleared
	m2 := GetIntBoolMap()
	if len(m2) != 0 {
		t.Errorf("expected cleared map, got length %d", len(m2))
	}
	PutIntBoolMap(m2)
}

func BenchmarkBufferPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := GetBuffer()
			buf.WriteString("test data for benchmarking")
			PutBuffer(buf)
		}
	})
	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.WriteString("test data for benchmarking")
		}
	})
}

func BenchmarkIntSlicePool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := GetIntSlice()
			*slice = append(*slice, 1, 2, 3, 4, 5)
			PutIntSlice(slice)
		}
	})
	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slice := make([]int, 0, 32)
			slice = append(slice, 1, 2, 3, 4, 5)
			_ = slice
		}
	})
}

func BenchmarkMapPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := GetIntBoolMap()
			m[1] = true
			m[2] = true
			m[3] = true
			PutIntBoolMap(m)
		}
	})
	b.Run("WithoutPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[int]bool, 16)
			m[1] = true
			m[2] = true
			m[3] = true
			_ = m
		}
	})
}
