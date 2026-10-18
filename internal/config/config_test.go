package config

import "testing"

func TestMust(t *testing.T) {
	if got := Must(Default()); got != Default() {
		t.Fatalf("Must changed configuration: %+v", got)
	}
	invalid := Default()
	invalid.Database = ""
	defer func() {
		if got := recover(); got != "config: initialization failed" {
			t.Fatalf("panic = %v", got)
		}
	}()
	Must(invalid)
	t.Fatal("Must did not panic")
}
