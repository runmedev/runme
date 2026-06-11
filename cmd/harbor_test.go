package cmd

import "testing"

func TestHarborCommandShape(t *testing.T) {
	root := Root()
	harbor, _, err := root.Find([]string{"harbor"})
	if err != nil {
		t.Fatal(err)
	}
	if harbor == nil {
		t.Fatal("harbor command not found")
	}
	if !harbor.Hidden {
		t.Fatal("harbor command is visible, want hidden")
	}

	stdio, _, err := root.Find([]string{"harbor", "stdio"})
	if err != nil {
		t.Fatal(err)
	}
	if stdio == nil {
		t.Fatal("harbor stdio command not found")
	}
	if !stdio.Hidden {
		t.Fatal("harbor stdio command is visible, want hidden")
	}
	if stdio.Use != "stdio" {
		t.Fatalf("stdio Use = %q, want stdio", stdio.Use)
	}
}
