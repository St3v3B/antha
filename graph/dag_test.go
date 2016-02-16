package graph

import (
	"fmt"
	"testing"
)

func checkEqual(expected []string, actual []Node) error {
	if e, a := len(expected), len(actual); e != a {
		return fmt.Errorf("expected %d elements found %d", e, a)
	}
	for i, e := range expected {
		if a, ok := actual[i].(string); !ok {

		} else if e != a {
			return fmt.Errorf("expected %q found %q", e, a)
		}
	}
	return nil
}

func TestIsNotDag(t *testing.T) {
	g := MakeTestGraph(map[string][]string{
		"a": []string{"b", "c"},
		"b": []string{"d"},
		"c": []string{"d"},
		"d": []string{"a"},
	})
	if err := IsDag(g); err == nil {
		t.Fatalf("failed to detect cycle")
	}
}

func TestTopoOrder(t *testing.T) {
	g := MakeTestGraph(map[string][]string{
		"a": []string{"b", "c"},
		"b": []string{"d"},
		"c": []string{"d"},
	})

	if order, err := TopoSort(TopoSortOpt{
		Graph: g,
		NodeOrder: func(a Node, b Node) bool {
			return a.(string) < b.(string)
		},
	}); err != nil {
		t.Fatalf("failed to construct DAG: %s", err)
	} else if err := checkEqual([]string{"d", "b", "c", "a"}, order); err != nil {
		t.Error(err)
	}
}

func TestTransitiveReduction(t *testing.T) {
	g := MakeTestGraph(map[string][]string{
		"a": []string{"b", "c", "d"},
		"b": []string{"c", "d"},
		"c": []string{"d"},
	})

	if gr, err := TransitiveReduction(g); err != nil {
		t.Error(err)
	} else if ng, nr := g.NumNodes(), gr.NumNodes(); ng != nr {
		t.Errorf("expected %d = %d", ng, nr)
	} else if l := gr.NumOuts("a"); l != 1 {
		t.Errorf("expected %d found %d", 1, l)
	} else if n := gr.Out("a", 0).(string); n != "b" {
		t.Errorf("expected %q found %q", "b", n)
	} else if l := gr.NumOuts("b"); l != 1 {
		t.Errorf("expected %d found %d", 1, l)
	} else if n := gr.Out("b", 0).(string); n != "c" {
		t.Errorf("expected %q found %q", "c", n)
	} else if l := gr.NumOuts("c"); l != 1 {
		t.Errorf("expected %d found %d", 1, l)
	} else if n := gr.Out("c", 0).(string); n != "d" {
		t.Errorf("expected %q found %q", "d", n)
	} else if l := gr.NumOuts("d"); l != 0 {
		t.Errorf("expected %d found %d", 0, l)
	}
}
