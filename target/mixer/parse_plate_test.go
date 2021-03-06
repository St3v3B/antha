package mixer

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/antha-lang/antha/antha/anthalib/wtype"
	"github.com/antha-lang/antha/inventory/testinventory"
)

func nonEmpty(m map[string]*wtype.LHWell) map[string]*wtype.LHComponent {
	r := make(map[string]*wtype.LHComponent)
	for addr, c := range m {
		if c.WContents.IsZero() {
			continue
		}
		r[addr] = c.WContents
	}
	return r
}

func getComponentsFromPlate(plate *wtype.LHPlate) []*wtype.LHComponent {

	var components []*wtype.LHComponent
	allWellPositions := plate.AllWellPositions(false)

	for _, wellcontents := range allWellPositions {

		if !plate.WellMap()[wellcontents].Empty() {

			component := plate.WellMap()[wellcontents].WContents
			components = append(components, component)

		}
	}
	return components
}

func allComponentsHaveWellLocation(plate *wtype.LHPlate) error {
	components := getComponentsFromPlate(plate)
	var errs []string
	for _, component := range components {
		if len(component.WellLocation()) == 0 {
			errs = append(errs, fmt.Errorf("no well location for %s after returning components from plate", component.Name()).Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func samePlate(a, b *wtype.LHPlate) error {
	if a.Type != b.Type {
		return fmt.Errorf("different types %q != %q", a.Type, b.Type)
	}
	compsA := nonEmpty(a.Wellcoords)
	compsB := nonEmpty(b.Wellcoords)

	if numA, numB := len(compsA), len(compsB); numA != numB {
		return fmt.Errorf("different number of non-empty wells %d != %d", numA, numB)
	}

	for addr, compA := range compsA {

		compB, ok := compsB[addr]
		if !ok {
			return fmt.Errorf("missing component in well %q", addr)
		}

		volA, volB := compA.Vol, compB.Vol
		if volA != volB {
			return fmt.Errorf("different volume in well %q: %f != %f", addr, volA, volB)
		}
		vunitA, vunitB := compA.Vunit, compB.Vunit
		if vunitA != vunitB && volA != 0.0 {
			return fmt.Errorf("different volume unit in well %q: %s != %s", addr, vunitA, vunitB)
		}
		concA, concB := compA.Conc, compB.Conc
		if concA != concB {
			return fmt.Errorf("different concentration in well %q: expected: %f; found: %f", addr, concA, concB)
		}
		cunitA, cunitB := compA.Cunit, compB.Cunit
		if cunitA != cunitB && concA != 0.0 {
			return fmt.Errorf("different concetration unit in well %q: expected: %s; found: %s", addr, cunitA, cunitB)
		}
	}

	return nil
}

func containsInvalidCharWarning(warnings []string) bool {
	for _, v := range warnings {
		if strings.Contains(v, "contains an invalid character \"+\"") {
			return true
		}
	}

	return false
}

func TestParsePlateWithValidation(t *testing.T) {
	ctx := testinventory.NewContext(context.Background())

	file := []byte(
		`
pcrplate_with_cooler,
A1,water+soil,water,50.0,ul,0,g/l,
A4,tea,water,50.0,ul,0,g/l,
A5,milk,water,100.0,ul,0,g/l,
`)
	r, err := ParsePlateCSVWithValidationConfig(ctx, bytes.NewBuffer(file), DefaultValidationConfig())

	if err != nil {
		t.Errorf("Failed to parse plate: %s ", err.Error())
	}
	if !containsInvalidCharWarning(r.Warnings) {
		t.Errorf("Default validation config must forbid + signs in component names")
	}
	r, err = ParsePlateCSVWithValidationConfig(ctx, bytes.NewBuffer(file), PermissiveValidationConfig())

	if err != nil {
		t.Errorf("Failed to parse plate: %s ", err.Error())
	}

	if containsInvalidCharWarning(r.Warnings) {
		t.Errorf("Permissive validation config must allow + signs in component names")
	}
}

func TestParsePlate(t *testing.T) {
	type testCase struct {
		File       []byte
		Expected   *wtype.LHPlate
		NoWarnings bool
	}

	ctx := testinventory.NewContext(context.Background())

	suite := []testCase{
		testCase{
			File: []byte(
				`
pcrplate_with_cooler,
A1,water,water,50.0,ul,0,g/l,
A4,tea,water,50.0,ul,10.0,mM/l,
A5,milk,water,100.0,ul,10.0,g/l,
A6,,,0,ul,0,g/l,
`),
			Expected: &wtype.LHPlate{
				Type: "pcrplate_with_cooler",
				Wellcoords: map[string]*wtype.LHWell{
					"A1": &wtype.LHWell{
						WContents: &wtype.LHComponent{
							CName: "water",
							Type:  wtype.LTWater,
							Vol:   50.0,
							Vunit: "ul",
							Conc:  0.0,
							Cunit: "g/l",
						},
					},
					"A4": &wtype.LHWell{
						WContents: &wtype.LHComponent{
							CName: "tea",
							Type:  wtype.LTWater,
							Vol:   50.0,
							Vunit: "ul",
							Conc:  10.0,
							Cunit: "mM/l",
						},
					},
					"A5": &wtype.LHWell{
						WContents: &wtype.LHComponent{
							CName: "milk",
							Type:  wtype.LTWater,
							Vol:   100.0,
							Vunit: "ul",
							Conc:  10.0,
							Cunit: "g/l",
						},
					},
				},
			},
		},
		testCase{
			File: []byte(
				`
pcrplate_skirted_riser40,Input_plate_1,LiquidType,Vol,Vol Unit,Conc,Conc Unit
A1,water,water,140.5,ul,0,mg/l
C1,neb5compcells,culture,20.5,ul,0,ng/ul
`),
			NoWarnings: true,
			Expected: &wtype.LHPlate{
				Type: "pcrplate_skirted_riser40",
				Wellcoords: map[string]*wtype.LHWell{
					"A1": &wtype.LHWell{
						WContents: &wtype.LHComponent{
							CName: "water",
							Type:  wtype.LTWater,
							Vol:   140.5,
							Vunit: "ul",
							Conc:  0,
							Cunit: "mg/l",
						},
					},
					"C1": &wtype.LHWell{
						WContents: &wtype.LHComponent{
							CName: "neb5compcells",
							Type:  wtype.LTCulture,
							Vol:   20.5,
							Vunit: "ul",
							Conc:  0,
							Cunit: "mg/l",
						},
					},
				},
			},
		},
	}

	for _, tc := range suite {
		p, err := ParsePlateCSV(ctx, bytes.NewBuffer(tc.File))
		if err != nil {
			t.Error(err)
		}
		if err := samePlate(tc.Expected, p.Plate); err != nil {
			t.Error(err)
		}
		if tc.NoWarnings && len(p.Warnings) != 0 {
			t.Errorf("found warnings: %s", p.Warnings)
		}

		if err := allComponentsHaveWellLocation(p.Plate); err != nil {
			t.Error(err.Error())
		}
	}
}
