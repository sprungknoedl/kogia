package main

import "testing"

func TestRequired(t *testing.T) {
	t.Run("nothing", requiredHelper([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 0))
	t.Run("almost", requiredHelper([]int{0, 0, 0, 0, 0, 0, 0, 1, 1, 1}, 0))
	t.Run("lowline", requiredHelper([]int{0, 0, 0, 1, 1, 1, 1, 1, 1, 1}, 1))
	t.Run("linear-up", requiredHelper([]int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90}, 3))
	t.Run("linear-down", requiredHelper([]int{90, 80, 70, 60, 50, 40, 30, 20, 10, 0}, 3))
	t.Run("spikes", requiredHelper([]int{1, 90, 90, 10, 10, 1, 1, 1, 1, 1}, 1))
	t.Run("sudden-jump", requiredHelper([]int{0, 0, 0, 60, 80, 90, 90, 90, 90, 90}, 6))
}

func requiredHelper(samples []int, expected int) func(t *testing.T) {
	return func(t *testing.T) {
		got := required(samples, .7, 10)
		if got != expected {
			t.Errorf("expected %d, got %d", expected, got)
		}
	}
}

func TestBound(t *testing.T) {
	t.Run("", boundHelper(0, 0, 0, 0))
	t.Run("", boundHelper(0, 5, 2, 2))
	t.Run("", boundHelper(1, 5, 0, 1))
	t.Run("", boundHelper(1, 10, 0, 1))
	t.Run("", boundHelper(1, 10, 14, 10))

	t.Run("upper-bound-precedence", boundHelper(20, 10, 12, 10))
}

func boundHelper(min, max, num, expected int) func(t *testing.T) {
	return func(t *testing.T) {
		got := bound(num, min, max)
		if got != expected {
			t.Errorf("expected %d, got %d", expected, got)
		}
	}
}
