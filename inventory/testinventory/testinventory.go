package testinventory

import (
	"context"
	"fmt"
	"sort"

	"github.com/antha-lang/antha/antha/anthalib/wtype"
	"github.com/antha-lang/antha/inventory"
)

type testInventory struct {
	componentByName map[string]*wtype.LHComponent
	plateByType     map[string]*wtype.LHPlate
	tipboxByType    map[string]*wtype.LHTipbox
	tipwasteByType  map[string]*wtype.LHTipwaste
}

func (i *testInventory) NewComponent(ctx context.Context, name string) (*wtype.LHComponent, error) {
	c, ok := i.componentByName[name]
	if !ok {
		return nil, inventory.ErrUnknownType
	}
	return c.Dup(), nil
}

func (i *testInventory) NewPlate(ctx context.Context, typ string) (*wtype.LHPlate, error) {
	p, ok := i.plateByType[typ]
	if !ok {
		return nil, inventory.ErrUnknownType
	}
	return p.Dup(), nil
}

func (i *testInventory) NewTipbox(ctx context.Context, typ string) (*wtype.LHTipbox, error) {
	tb, ok := i.tipboxByType[typ]
	if !ok {
		return nil, inventory.ErrUnknownType
	}
	return tb.Dup(), nil
}

func (i *testInventory) NewTipwaste(ctx context.Context, typ string) (*wtype.LHTipwaste, error) {
	tw, ok := i.tipwasteByType[typ]
	if !ok {
		return nil, inventory.ErrUnknownType
	}
	return tw.Dup(), nil
}

func (i *testInventory) XXXGetPlates(ctx context.Context) ([]*wtype.LHPlate, error) {
	plates := GetPlates(ctx)
	return plates, nil
}

// NewContext creates a new test inventory context
func NewContext(ctx context.Context) context.Context {
	inv := &testInventory{
		componentByName: make(map[string]*wtype.LHComponent),
		plateByType:     make(map[string]*wtype.LHPlate),
		tipboxByType:    make(map[string]*wtype.LHTipbox),
		tipwasteByType:  make(map[string]*wtype.LHTipwaste),
	}

	for _, c := range makeComponents() {
		if _, seen := inv.componentByName[c.CName]; seen {
			panic(fmt.Sprintf("component %s already added", c.CName))
		}
		inv.componentByName[c.CName] = c
	}

	for _, p := range makePlates() {
		if _, seen := inv.plateByType[p.Type]; seen {
			panic(fmt.Sprintf("plate %s already added", p.Type))
		}
		inv.plateByType[p.Type] = p
	}

	for _, tb := range makeTipboxes() {
		if _, seen := inv.tipboxByType[tb.Type]; seen {
			panic(fmt.Sprintf("tipbox %s already added", tb.Type))
		}
		if _, seen := inv.tipboxByType[tb.Tiptype.Type]; seen {
			panic(fmt.Sprintf("tipbox %s already added", tb.Tiptype.Type))
		}
		inv.tipboxByType[tb.Type] = tb
		inv.tipboxByType[tb.Tiptype.Type] = tb
	}

	for _, tw := range makeTipwastes() {
		if _, seen := inv.tipwasteByType[tw.Type]; seen {
			panic(fmt.Sprintf("tipwaste %s already added", tw.Type))
		}
		inv.tipwasteByType[tw.Type] = tw
	}

	return inventory.NewContext(ctx, inv)
}

// GetTipboxes returns the tipboxes in a test inventory context
func GetTipboxes(ctx context.Context) []*wtype.LHTipbox {
	inv := inventory.GetInventory(ctx).(*testInventory)
	var tbs []*wtype.LHTipbox
	for _, tb := range inv.tipboxByType {
		tbs = append(tbs, tb)
	}

	sort.Slice(tbs, func(i, j int) bool {
		return tbs[i].Type < tbs[j].Type
	})

	return tbs
}

// GetPlates returns the plates in a test inventory context
func GetPlates(ctx context.Context) []*wtype.LHPlate {
	inv := inventory.GetInventory(ctx).(*testInventory)
	var ps []*wtype.LHPlate
	for _, p := range inv.plateByType {
		ps = append(ps, p)
	}

	sort.Slice(ps, func(i, j int) bool {
		return ps[i].Type < ps[j].Type
	})

	return ps
}

// GetComponents returns the components in a test inventory context
func GetComponents(ctx context.Context) []*wtype.LHComponent {
	inv := inventory.GetInventory(ctx).(*testInventory)
	var cs []*wtype.LHComponent
	for _, c := range inv.componentByName {
		cs = append(cs, c)
	}

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Type < cs[j].Type
	})

	return cs
}
