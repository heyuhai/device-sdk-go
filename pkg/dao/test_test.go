package dao

import "testing"

func TestAppend(t *testing.T) {
	foo := []string{"1", "3", "5"}
	bar := []string{"c", "a", "b"}
	merge := append(foo, bar...)
	t.Log(merge)
	t.Log(merge[3])
}
