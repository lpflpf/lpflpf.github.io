package main

import "testing"

func BenchmarkOttoAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RunOtto(1, 2)
	}
}
func BenchmarkGojaAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RunGoja(1, 2)
	}
}
func BenchmarkV8Add(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RunV8(1, 2)
	}
}

func BenchmarkSumOtto(b *testing.B) {
	for i := 0; i < b.N; i++ {
		OttoSum()
	}
}

func BenchmarkSumGoja(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GojaSum()
	}
}

func BenchmarkSumV8(b *testing.B) {
	for i := 0; i < b.N; i++ {
		V8Sum()
	}
}
