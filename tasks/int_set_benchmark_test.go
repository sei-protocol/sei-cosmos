package tasks

import (
	"math/rand"
	"testing"
)

func intSetImpl(size int) IntSet {
	return newSyncSet(size)
}

func BenchmarkSyncSet_Add(b *testing.B) {
	size := 1000 // Assuming a size of 1000 for this example
	ss := intSetImpl(size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ss.Add(i % size) // Loop over the syncSet size to avoid out-of-range panics
	}
}

func BenchmarkSyncSet_Delete(b *testing.B) {
	size := 1000
	ss := intSetImpl(size)
	// Pre-fill the syncSet to delete from
	for i := 0; i < size; i++ {
		ss.Add(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ss.Delete(i % size) // Loop over the syncSet size to avoid out-of-range panics
	}
}

func BenchmarkSyncSet_Length(b *testing.B) {
	size := 1000
	ss := intSetImpl(size)
	// Pre-fill the syncSet
	for i := 0; i < size; i++ {
		ss.Add(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ss.Length()
	}
}

func BenchmarkSyncSet_Exists(b *testing.B) {
	size := 1000
	ss := intSetImpl(size)
	// Pre-fill the syncSet
	for i := 0; i < size; i++ {
		ss.Add(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ss.Exists(i % size) // Loop over the syncSet size to avoid out-of-range panics
	}
}

func BenchmarkSyncSet_Add_Contention(b *testing.B) {
	size := 1000 // The size of the syncSet
	ss := intSetImpl(size)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ss.Add(rand.Intn(size)) // Use a random index for contention
		}
	})
}

func BenchmarkSyncSet_Delete_Contention(b *testing.B) {
	size := 1000 // The size of the syncSet
	ss := intSetImpl(size)
	// Pre-fill the syncSet to delete from
	for i := 0; i < size; i++ {
		ss.Add(i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ss.Delete(rand.Intn(size)) // Use a random index for contention
		}
	})
}

func BenchmarkSyncSet_Exists_Contention(b *testing.B) {
	size := 1000 // The size of the syncSet
	ss := intSetImpl(size)
	// Pre-fill the syncSet
	for i := 0; i < size; i++ {
		ss.Add(i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ss.Exists(rand.Intn(size)) // Use a random index for contention
		}
	})
}
