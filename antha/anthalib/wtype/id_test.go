package wtype

import (
	"github.com/antha-lang/antha/antha/anthalib/wutil"
	"testing"
)

// tests on ID arithmetic

func makeWell() *LHWell {
	swshp := NewShape("box", "mm", 8.2, 8.2, 41.3)
	welltype := NewLHWell("DSW96", "", "", "ul", 1000, 100, swshp, LHWBV, 8.2, 8.2, 41.3, 4.7, "mm")
	welltype.WContents.Loc = "randomplate:A1"
	return welltype
}

func makeComponent() *LHComponent {
	rand := wutil.GetRandom()
	A := NewLHComponent()
	A.Type = LTWater
	A.Smax = 9999
	A.Vol = rand.Float64() * 10.0
	A.Vunit = "ul"
	A.Loc = "anotherrandomplate:A2"
	return A
}

// mix to empty well should result in component
// in well with new ID
// well component should have old component as parent
func TestEmptyWellMix(t *testing.T) {
	c := makeComponent()
	w := makeWell()
	w.Add(c)

	if w.WContents.ID == c.ID {
		t.Fatal("Well contents should have different ID to input component")
	}

	if !w.WContents.HasParent(c.ID) {
		t.Fatal("Well contents should have input component ID as parent")
	}
	if !c.HasDaughter(w.WContents.ID) {
		t.Fatal("Component mixed into well should have well contents as daughter")
	}
}

func TestFullWellMix(t *testing.T) {
	c := makeComponent()
	w := makeWell()
	idb4 := w.WContents.ID
	w.Add(c)
	if w.WContents.HasParent(w.WContents.ID) {
		t.Fatal("Components should not have themselves as parents! It's just too metaphysical")
	}
	d := makeComponent()
	w.Add(d)
	if w.WContents.ID == c.ID || w.WContents.ID == d.ID || w.WContents.ID == idb4 {
		t.Fatal("Well contents should have new ID after mix")
	}
	if !w.WContents.HasParent(c.ID) || !w.WContents.HasParent(d.ID) {
		t.Fatal("Well contents should have all parents set")
	}

	if !d.HasDaughter(w.WContents.ID) {
		t.Fatal("Component mixed into well should have well contents as daughter")
	}

	if w.WContents.HasParent(w.WContents.ID) {
		t.Fatal("Components should not have themselves as parents! It's just too metaphysical")
	}

	e := makeComponent()

	w.Add(e)

	f := makeComponent()

	w.Add(f)

	w2 := makeWell()

	g := makeComponent()

	w2.Add(w.WContents)
	w2.Add(g)

	if !w2.WContents.HasParent(c.ID) || !w2.WContents.HasParent(d.ID) || !w2.WContents.HasParent(e.ID) || !w2.WContents.HasParent(f.ID) || !w2.WContents.HasParent(w.WContents.ID) {
		t.Fatal("Well contents should have all parents set...2")
	}

	/*
		gra := w2.WContents.ParentTree()
		fmt.Println(w2.WContents.ParentID)
		fmt.Println(gra.Nodes)
		for n, a := range gra.Outs {
			fmt.Println(n, ":::", a)
		}

		s := graph.Print(graph.PrintOpt{Graph: &gra})

		fmt.Println(s)
	*/
}
