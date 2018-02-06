// liquidhandling/lhtypes.Go: Part of the Antha language
// Copyright (C) 2014 the Antha authors. All rights reserved.
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
//
// For more information relating to the software or licensing issues please
// contact license@antha-lang.Org or write to the Antha team c/o
// Synthace Ltd. The London Bioscience Innovation Centre
// 2 Royal College St, London NW1 0NH UK

// defines types for dealing with liquid handling requests
package wtype

import (
	"encoding/csv"
	"fmt"
	"github.com/antha-lang/antha/antha/anthalib/wunit"
	"github.com/antha-lang/antha/antha/anthalib/wutil"
	"github.com/antha-lang/antha/microArch/logger"
	"math"
	"os"
	"strconv"
	"time"
)

// structure describing a microplate
type LHPlate struct {
	ID          string
	Inst        string
	Loc         string             // location of plate
	PlateName   string             // user-definable plate name
	Type        string             // plate type
	Mnfr        string             // manufacturer
	WlsX        int                // wells along long axis
	WlsY        int                // wells along short axis
	Nwells      int                // total number of wells
	HWells      map[string]*LHWell // map of well IDs to well
	Rows        [][]*LHWell
	Cols        [][]*LHWell
	Welltype    *LHWell
	Wellcoords  map[string]*LHWell // map of coords in A1 format to wells
	WellXOffset float64            // distance (mm) between well centres in X direction
	WellYOffset float64            // distance (mm) between well centres in Y direction
	WellXStart  float64            // offset (mm) to first well in X direction
	WellYStart  float64            // offset (mm) to first well in Y direction
	WellZStart  float64            // offset (mm) to bottom of well in Z direction
	Bounds      BBox               // (relative) position of the plate (mm), set by parent
	parent      LHObject           `gotopb:"-"`
}

func (lhp LHPlate) String() string {
	return fmt.Sprintf(
		`LHPlate {
	ID          : %s,
	Inst        : %s,
	Loc         : %s,
	PlateName   : %s,
	Type        : %s,
	Mnfr        : %s,
	WlsX        : %d,
	WlsY        : %d,
	Nwells      : %d,
	HWells      : %p,
	Rows        : %p,
	Cols        : %p,
	Welltype    : %p,
	Wellcoords  : %p,
	WellXOffset : %f,
	WellYOffset : %f,
	WellXStart  : %f,
	WellYStart  : %f,
	WellZStart  : %f,
	Size  : %f x %f x %f,
}`,
		lhp.ID,
		lhp.Inst,
		lhp.Loc,
		lhp.PlateName,
		lhp.Type,
		lhp.Mnfr,
		lhp.WlsX,
		lhp.WlsY,
		lhp.Nwells,
		lhp.HWells,
		lhp.Rows,
		lhp.Cols,
		lhp.Welltype,
		lhp.Wellcoords,
		lhp.WellXOffset,
		lhp.WellYOffset,
		lhp.WellXStart,
		lhp.WellYStart,
		lhp.WellZStart,
		lhp.Bounds.GetSize().X,
		lhp.Bounds.GetSize().Y,
		lhp.Bounds.GetSize().Z,
	)
}

// convenience method

func (lhp *LHPlate) GetComponent(cmp *LHComponent, exact bool, mpv wunit.Volume) ([]WellCoords, []wunit.Volume, bool) {
	ret := make([]WellCoords, 0, 1)
	vols := make([]wunit.Volume, 0, 1)
	it := NewOneTimeColumnWiseIterator(lhp)

	var volGot wunit.Volume
	volGot = wunit.NewVolume(0.0, "ul")
	volWant := cmp.Volume().Dup()

	x := 0

	for wc := it.Curr(); it.Valid(); wc = it.Next() {
		w := lhp.Wellcoords[wc.FormatA1()]

		/*
			if !w.Empty() {
				logger.Debug(fmt.Sprint("WANT: ", cmp.CName, " :: ", wc.FormatA1(), " ", w.Contents().CName, " ", w.CurrVolume().ToString()))
			}
		*/
		if w.Contents().CName == cmp.CName {
			if exact && w.Contents().ID != cmp.ID {
				continue
			}
			x += 1

			v := w.WorkingVolume()
			if v.LessThan(mpv) {
				continue
			}
			volGot.Add(v)
			ret = append(ret, wc)

			if volWant.GreaterThan(v) {
				vols = append(vols, v)
			} else {
				vols = append(vols, volWant.Dup())
			}

			volWant.Subtract(v)

			if volGot.GreaterThan(cmp.Volume()) || volGot.EqualTo(cmp.Volume()) {
				break
			}
		}
	}

	//fmt.Println("FOUND: ", cmp.CName, " WANT ", cmp.Volume().ToString(), " GOT ", volGot.ToString(), "  ", ret)

	if !(volGot.GreaterThan(cmp.Volume()) || volGot.EqualTo(cmp.Volume())) {
		return ret, vols, false
	}

	return ret, vols, true
}

func (lhp *LHPlate) Wells() [][]*LHWell {
	return lhp.Rows
}
func (lhp *LHPlate) WellMap() map[string]*LHWell {
	return lhp.Wellcoords
}

func (lhp *LHPlate) AllWellPositions() (wellpositionarray []string) {

	wellpositionarray = make([]string, 0)

	// range through well coordinates
	for j := 0; j < lhp.WlsX; j++ {
		for i := 0; i < lhp.WlsY; i++ {
			wellposition := wutil.NumToAlpha(i+1) + strconv.Itoa(j+1)
			wellpositionarray = append(wellpositionarray, wellposition)
		}
	}
	return
}

// @implement named

func (lhp *LHPlate) GetName() string {
	if lhp == nil {
		return "<nil>"
	}
	return lhp.PlateName
}

// @implement Typed
func (lhp *LHPlate) GetType() string {
	if lhp == nil {
		return "<nil>"
	}
	return lhp.Type
}

func (self *LHPlate) GetClass() string {
	return "plate"
}

func (lhp *LHPlate) WellAt(wc WellCoords) *LHWell {
	return lhp.Wellcoords[wc.FormatA1()]
}

func (lhp *LHPlate) WellAtString(s string) (*LHWell, bool) {
	// improve later, start by assuming these are in FormatA1()
	w, ok := lhp.Wellcoords[s]

	return w, ok
}

func (lhp *LHPlate) WellsX() int {
	return lhp.WlsX
}

func (lhp *LHPlate) WellsY() int {
	return lhp.WlsY
}

func (lhp *LHPlate) NextEmptyWell(it PlateIterator) WellCoords {
	c := 0
	for wc := it.Curr(); it.Valid(); wc = it.Next() {
		if c == lhp.Nwells {
			// prevent iterators from ever making this loop infinitely
			break
		}

		if lhp.Cols[wc.X][wc.Y].Empty() {
			return wc
		}
	}

	return ZeroWellCoords()
}

func NewLHPlate(platetype, mfr string, nrows, ncols int, size Coordinates, welltype *LHWell, wellXOffset, wellYOffset, wellXStart, wellYStart, wellZStart float64) *LHPlate {
	var lhp LHPlate
	lhp.Type = platetype
	//lhp.ID = "plate-" + GetUUID()
	lhp.ID = GetUUID()
	lhp.PlateName = fmt.Sprintf("%s_%s", platetype, lhp.ID[1:len(lhp.ID)-2])
	lhp.Mnfr = mfr
	lhp.WlsX = ncols
	lhp.WlsY = nrows
	lhp.Nwells = ncols * nrows
	welltype.Plate = &lhp
	lhp.Welltype = welltype
	lhp.WellXOffset = wellXOffset
	lhp.WellYOffset = wellYOffset
	lhp.WellXStart = wellXStart
	lhp.WellYStart = wellYStart
	lhp.WellZStart = wellZStart
	lhp.Bounds.SetSize(size)

	wellcoords := make(map[string]*LHWell, ncols*nrows)

	// make wells
	rowarr := make([][]*LHWell, nrows)
	colarr := make([][]*LHWell, ncols)
	arr := make([][]*LHWell, nrows)
	wellmap := make(map[string]*LHWell, ncols*nrows)

	for i := 0; i < nrows; i++ {
		arr[i] = make([]*LHWell, ncols)
		rowarr[i] = make([]*LHWell, ncols)
		for j := 0; j < ncols; j++ {
			if colarr[j] == nil {
				colarr[j] = make([]*LHWell, nrows)
			}
			arr[i][j] = welltype.CDup()

			//crds := wutil.NumToAlpha(i+1) + ":" + strconv.Itoa(j+1)
			crds := WellCoords{j, i}
			wellcoords[crds.FormatA1()] = arr[i][j]
			colarr[j][i] = arr[i][j]
			rowarr[i][j] = arr[i][j]
			wellmap[arr[i][j].ID] = arr[i][j]
			arr[i][j].Plate = &lhp
			arr[i][j].Crds = crds
			arr[i][j].WContents.Loc = lhp.ID + ":" + crds.FormatA1()
			arr[i][j].SetOffset(Coordinates{
				wellXStart + float64(j)*wellXOffset,
				wellYStart + float64(i)*wellYOffset,
				wellZStart,
			})
		}
	}

	lhp.Wellcoords = wellcoords
	lhp.HWells = wellmap
	lhp.Cols = colarr
	lhp.Rows = rowarr

	return &lhp
}

func (lhp *LHPlate) Dup() *LHPlate {
	// protect yourself fgs
	if lhp == nil {
		logger.Fatal(fmt.Sprintln("Can't dup nonexistent plate"))
	}
	ret := NewLHPlate(lhp.Type, lhp.Mnfr, lhp.WlsY, lhp.WlsX, lhp.GetSize(), lhp.Welltype, lhp.WellXOffset, lhp.WellYOffset, lhp.WellXStart, lhp.WellYStart, lhp.WellZStart)

	ret.PlateName = lhp.PlateName

	ret.HWells = make(map[string]*LHWell, len(ret.HWells))

	for i, row := range lhp.Rows {
		for j, well := range row {
			d := well.Dup()
			ret.Rows[i][j] = d
			ret.Cols[j][i] = d
			ret.Wellcoords[d.Crds.FormatA1()] = d
			ret.HWells[d.ID] = d
			d.WContents.Loc = ret.ID + ":" + d.Crds.FormatA1()
			d.Plate = ret
		}
	}

	return ret
}
func (lhp *LHPlate) DupKeepIDs() *LHPlate {
	// protect yourself fgs
	if lhp == nil {
		logger.Fatal(fmt.Sprintln("Can't dup nonexistent plate"))
	}
	ret := NewLHPlate(lhp.Type, lhp.Mnfr, lhp.WlsY, lhp.WlsX, lhp.GetSize(), lhp.Welltype, lhp.WellXOffset, lhp.WellYOffset, lhp.WellXStart, lhp.WellYStart, lhp.WellZStart)
	ret.ID = lhp.ID

	ret.PlateName = lhp.PlateName

	ret.HWells = make(map[string]*LHWell, len(ret.HWells))

	for i, row := range lhp.Rows {
		for j, well := range row {
			d := well.Dup()
			d.ID = well.ID
			ret.Rows[i][j] = d
			ret.Cols[j][i] = d
			ret.Wellcoords[d.Crds.FormatA1()] = d
			ret.HWells[d.ID] = d
			d.WContents.ID = well.WContents.ID
			d.WContents.Loc = ret.ID + ":" + d.Crds.FormatA1()
			d.Plate = ret
		}
	}

	return ret
}

func (p *LHPlate) ProtectAllWells() {
	for _, v := range p.Wellcoords {
		v.Protect()
	}
}

func (p *LHPlate) UnProtectAllWells() {
	for _, v := range p.Wellcoords {
		v.UnProtect()
	}
}

func Initialize_Wells(plate *LHPlate) {
	wells := (*plate).HWells
	newwells := make(map[string]*LHWell, len(wells))
	wellcrds := (*plate).Wellcoords
	for _, well := range wells {
		well.ID = GetUUID()
		newwells[well.ID] = well
		wellcrds[well.Crds.FormatA1()] = well
	}
	(*plate).HWells = newwells
	(*plate).Wellcoords = wellcrds
}

func (p *LHPlate) RemoveComponent(well string, vol wunit.Volume) *LHComponent {
	w := p.Wellcoords[well]

	if w == nil {
		logger.Debug(fmt.Sprint("RemoveComponent (plate) ERROR: ", well, " ", vol.ToString(), " Can't find well"))
		return nil
	}

	c, _ := w.Remove(vol)

	return c
}

func (p *LHPlate) DeclareTemporary() {
	for _, w := range p.Wellcoords {
		w.DeclareTemporary()
	}
}

func (p *LHPlate) IsTemporary() bool {
	for _, w := range p.Wellcoords {
		if !w.IsTemporary() {
			return false
		}
	}

	return true
}

func (p *LHPlate) DeclareAutoallocated() {
	for _, w := range p.Wellcoords {
		w.DeclareAutoallocated()
	}
}

func (p *LHPlate) IsAutoallocated() bool {
	for _, w := range p.Wellcoords {
		if !w.IsAutoallocated() {
			return false
		}
	}

	return true
}

func ExportPlateCSV(outputpilename string, plate *LHPlate, platename string, wells []string, liquids []*LHComponent, Volumes []wunit.Volume) error {

	csvfile, err := os.Create(outputpilename)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}
	defer csvfile.Close()

	records := make([][]string, 0)

	//record := make([]string, 0)

	headerrecord := []string{plate.Type, platename, "", "", ""}

	records = append(records, headerrecord)

	for i, well := range wells {

		volfloat := Volumes[i].RawValue()

		volstr := strconv.FormatFloat(volfloat, 'G', -1, 64)

		record := []string{well, liquids[i].CName, liquids[i].TypeName(), volstr, Volumes[i].Unit().PrefixedSymbol()}
		records = append(records, record)
	}

	csvwriter := csv.NewWriter(csvfile)

	for _, record := range records {

		err = csvwriter.Write(record)

		if err != nil {
			return err
		}
	}
	csvwriter.Flush()

	return err
}
func (p *LHPlate) SetConstrained(platform string, positions []string) {
	p.Welltype.Extra[platform] = positions
}

func (p *LHPlate) IsConstrainedOn(platform string) ([]string, bool) {
	var pos []string

	par, ok := p.Welltype.Extra[platform]

	if ok {
		pos = par.([]string)
		return pos, true
	}

	return pos, false

}

//##############################################
//@implement LHObject
//##############################################

func (self *LHPlate) GetPosition() Coordinates {
	if self.parent != nil {
		return self.parent.GetPosition().Add(self.Bounds.GetPosition())
	}
	return self.Bounds.GetPosition()
}

func (self *LHPlate) GetSize() Coordinates {
	return self.Bounds.GetSize()
}

func (self *LHPlate) GetWellBounds() BBox {
	return BBox{
		self.Bounds.GetPosition().Add(Coordinates{self.WellXStart, self.WellYStart, self.WellZStart}),
		Coordinates{self.WellXOffset * float64(self.NCols()), self.WellYOffset * float64(self.NRows()), self.Welltype.GetSize().Z},
	}
}

func (self *LHPlate) GetBoxIntersections(box BBox) []LHObject {
	//relative to me
	box.SetPosition(box.GetPosition().Subtract(OriginOf(self)))
	ret := []LHObject{}
	if self.Bounds.IntersectsBox(box) {
		ret = append(ret, self)
	}

	if self.GetWellBounds().IntersectsBox(box) {
		for _, row := range self.Rows {
			for _, well := range row {
				ret = append(ret, well.GetBoxIntersections(box)...)
			}
		}
	}
	//todo, scan through wells
	return ret
}

func (self *LHPlate) GetPointIntersections(point Coordinates) []LHObject {
	//relative
	point = point.Subtract(OriginOf(self))
	ret := []LHObject{}

	if self.GetWellBounds().IntersectsPoint(point) {
		for _, row := range self.Rows {
			for _, well := range row {
				ret = append(ret, well.GetPointIntersections(point)...)
			}
		}
	}

	if len(ret) == 0 && self.Bounds.IntersectsPoint(point) {
		ret = append(ret, self)
	}
	return ret
}

func (p *LHPlate) Evaporate(time time.Duration, env Environment) []VolumeCorrection {
	ret := make([]VolumeCorrection, 0, 10)
	if p == nil {
		return ret
	}
	for _, w := range p.Wellcoords {
		if !w.Empty() {
			vc := w.Evaporate(time, env)
			if vc.Type != "" {
				ret = append(ret, vc)
			}
		}
	}
	return ret
}

func (self *LHPlate) SetOffset(o Coordinates) error {
	self.Bounds.SetPosition(o)
	return nil
}

func (self *LHPlate) SetParent(p LHObject) error {
	self.parent = p
	return nil
}

func (self *LHPlate) GetParent() LHObject {
	return self.parent
}

//##############################################
//@implement Addressable
//##############################################

func (self *LHPlate) AddressExists(c WellCoords) bool {
	return c.X >= 0 &&
		c.Y >= 0 &&
		c.X < self.WlsX &&
		c.Y < self.WlsY
}

func (lhp *LHPlate) NCols() int {
	return lhp.WlsX
}

func (lhp *LHPlate) NRows() int {
	return lhp.WlsY
}

func (self *LHPlate) GetChildByAddress(c WellCoords) LHObject {
	if !self.AddressExists(c) {
		return nil
	}
	//LHWells aren't LHObjects yet
	return self.Cols[c.X][c.Y]
}

func (self *LHPlate) CoordsToWellCoords(r Coordinates) (WellCoords, Coordinates) {
	rel := r.Subtract(self.GetPosition())
	wc := WellCoords{
		int(math.Floor(((rel.X - self.WellXStart) / self.WellXOffset))), // + 0.5), Don't need to add .5 because
		int(math.Floor(((rel.Y - self.WellYStart) / self.WellYOffset))), // + 0.5), WellXStart is to edge, not center
	}
	if wc.X < 0 {
		wc.X = 0
	} else if wc.X >= self.WlsX {
		wc.X = self.WlsX - 1
	}
	if wc.Y < 0 {
		wc.Y = 0
	} else if wc.Y >= self.WlsY {
		wc.Y = self.WlsY - 1
	}

	r2, _ := self.WellCoordsToCoords(wc, TopReference)

	return wc, r.Subtract(r2)
}

func (self *LHPlate) WellCoordsToCoords(wc WellCoords, r WellReference) (Coordinates, bool) {
	if !self.AddressExists(wc) {
		return Coordinates{}, false
	}

	var z float64
	if r == BottomReference {
		z = self.WellZStart
	} else if r == TopReference {
		z = self.WellZStart + self.Welltype.GetSize().Z
	} else if r == LiquidReference {
		panic("Haven't implemented liquid level yet")
	}

	return self.GetPosition().Add(Coordinates{
		self.WellXStart + (float64(wc.X)+0.5)*self.WellXOffset,
		self.WellYStart + (float64(wc.Y)+0.5)*self.WellYOffset,
		z}), true
}

func (p *LHPlate) ResetID(newID string) {
	for _, w := range p.Wellcoords {
		w.ResetPlateID(newID)
	}
	p.ID = newID
}

func (p *LHPlate) Height() float64 {
	return p.Bounds.GetSize().Z
}

func (p *LHPlate) IsUserAllocated() bool {
	// true if any wells are user allocated

	for _, w := range p.Wellcoords {
		if w.IsUserAllocated() {
			return true
		}
	}

	return false
}

func (p *LHPlate) MergeWith(p2 *LHPlate) {
	// do nothing if these are not same type

	if p.Type != p2.Type {
		return
	}

	// transfer any non-User-Allocated wells in here

	it := NewOneTimeColumnWiseIterator(p)

	for ; it.Valid(); it.Next() {
		wc := it.Curr()

		if !it.Valid() {
			break
		}

		w1 := p.Wellcoords[wc.FormatA1()]
		w2 := p2.Wellcoords[wc.FormatA1()]

		if !w1.IsUserAllocated() {
			w1.WContents = w2.WContents
		}
	}
}

func (p *LHPlate) MarkNonEmptyWellsUserAllocated() {
	for _, w := range p.Wellcoords {
		if !w.Empty() {
			w.SetUserAllocated()
		}
	}
}
