// Package pool provides object pools for reducing memory allocations in hot paths.
package pool

import (
	"bytes"
	"strings"
	"sync"
)

// BufferPool is a pool of bytes.Buffer for reducing allocations in JSON serialization.
var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer returns a buffer from the pool.
func GetBuffer() *bytes.Buffer {
	buf := BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the pool.
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	// Prevent holding onto huge buffers
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	BufferPool.Put(buf)
}

// IntSlicePool is a pool of []int for channel selection.
var IntSlicePool = sync.Pool{
	New: func() interface{} {
		slice := make([]int, 0, 32)
		return &slice
	},
}

// GetIntSlice returns an int slice from the pool.
func GetIntSlice() *[]int {
	slice := IntSlicePool.Get().(*[]int)
	*slice = (*slice)[:0]
	return slice
}

// PutIntSlice returns an int slice to the pool.
func PutIntSlice(slice *[]int) {
	if slice == nil {
		return
	}
	if cap(*slice) > 1024 {
		return
	}
	*slice = (*slice)[:0]
	IntSlicePool.Put(slice)
}

// StringBuilderPool is a pool of strings.Builder for string concatenation.
var StringBuilderPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

// GetStringBuilder returns a string builder from the pool.
func GetStringBuilder() *strings.Builder {
	sb := StringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// PutStringBuilder returns a string builder to the pool.
func PutStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	// Don't pool very large builders
	if sb.Cap() > 64*1024 {
		return
	}
	sb.Reset()
	StringBuilderPool.Put(sb)
}

// MapPool provides a pool for map[int]bool used in channel selection.
var MapPool = sync.Pool{
	New: func() interface{} {
		return make(map[int]bool, 16)
	},
}

// GetIntBoolMap returns a map from the pool.
func GetIntBoolMap() map[int]bool {
	m := MapPool.Get().(map[int]bool)
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	return m
}

// PutIntBoolMap returns a map to the pool.
func PutIntBoolMap(m map[int]bool) {
	if m == nil {
		return
	}
	if len(m) > 1024 {
		return
	}
	MapPool.Put(m)
}

// GenericSlicePool is a pool for []interface{} slices.
var GenericSlicePool = sync.Pool{
	New: func() interface{} {
		slice := make([]interface{}, 0, 32)
		return &slice
	},
}

// GetGenericSlice returns a generic slice from the pool.
func GetGenericSlice() *[]interface{} {
	slice := GenericSlicePool.Get().(*[]interface{})
	*slice = (*slice)[:0]
	return slice
}

// PutGenericSlice returns a generic slice to the pool.
func PutGenericSlice(slice *[]interface{}) {
	if slice == nil {
		return
	}
	if cap(*slice) > 1024 {
		return
	}
	*slice = (*slice)[:0]
	GenericSlicePool.Put(slice)
}
