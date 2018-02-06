// liquidhandling/lhwell.Go: Part of the Antha language
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
	"fmt"
	"github.com/antha-lang/antha/antha/AnthaStandardLibrary/Packages/eng"
	"github.com/antha-lang/antha/antha/anthalib/wunit"
	"github.com/antha-lang/antha/antha/anthalib/wutil"
	"github.com/antha-lang/antha/microArch/logger"
	"strings"
	"time"
)

type WellBottomType int

const (
	FlatWellBottom WellBottomType = iota
	UWellBottom
	VWellBottom
)

var WellBottomNames []string = []string{"flat", "U", "V"}

// structure representing a well on a microplate - description of a destination
type LHWell struct {
	ID        string
	Inst      string
	Crds      WellCoords
	MaxVol    float64
	WContents *LHComponent
	Rvol      float64
	WShape    *Shape
	Bottom    WellBottomType
	Bounds    BBox
	Bottomh   float64
	Extra     map[string]interface{}
	Plate     LHObject `gotopb:"-" json:"-"`
}

//@implement Named
func (self *LHWell) GetName() string {
	return fmt.Sprintf("%s@%s", self.Crds.FormatA1(), NameOf(self.Plate))
}

//@implement Typed
func (self *LHWell) GetType() string {
	return fmt.Sprintf("well_in_%s", TypeOf(self.Plate))
}

//@implement Classy
func (self *LHWell) GetClass() string {
	return "well"
}

//@implement LHObject
func (self *LHWell) GetPosition() Coordinates {
	return OriginOf(self).Add(self.Bounds.GetPosition())
}

//@implement LHObject
func (self *LHWell) GetSize() Coordinates {
	return self.Bounds.GetSize()
}

//@implement LHObject
func (self *LHWell) GetBoxIntersections(box BBox) []LHObject {
	//relative box
	box.SetPosition(box.GetPosition().Subtract(OriginOf(self)))
	if self.Bounds.IntersectsBox(box) {
		return []LHObject{self}
	}
	return nil
}

//@implement LHObject
func (self *LHWell) GetPointIntersections(point Coordinates) []LHObject {
	//relative point
	point = point.Subtract(OriginOf(self))
	//At some point this should be called self.shape for a more accurate intersection test
	//see branch shape-changes
	if self.Bounds.IntersectsPoint(point) {
		return []LHObject{self}
	}
	return nil
}

//@implement LHObject
func (self *LHWell) SetOffset(point Coordinates) error {
	self.Bounds.SetPosition(point)
	return nil
}

//@implement LHObject
func (self *LHWell) SetParent(p LHObject) error {
	//Seems unlikely, but I suppose wells that you can take from one plate and insert
	//into another could be feasible with some funky labware
	if plate, ok := p.(*LHPlate); ok {
		self.Plate = plate
		return nil
	}
	if tb, ok := p.(*LHTipwaste); ok {
		self.Plate = tb
		return nil
	}
	return fmt.Errorf("Cannot set well parent to %s \"%s\", only plates allowed", ClassOf(p), NameOf(p))
}

//@implement LHObject
func (self *LHWell) GetParent() LHObject {
	return self.Plate
}

func (w LHWell) String() string {
	plate := w.Plate.(*LHPlate)
	return fmt.Sprintf(
		`LHWELL{
ID        : %s,
Inst      : %s,
Plateinst : %s,
Plateid   : %s,
Platetype : %s,
Crds      : %s,
MaxVol    : %g ul,
WContents : %v,
Rvol      : %g ul,
WShape    : %v,
Bottom    : %s,
size      : [%v x %v x %v]mm,
Bottomh   : %g,
Extra     : %v,
Plate     : %v,
}`,
		w.ID,
		w.Inst,
		plate.Inst,
		plate.ID,
		plate.GetType(),
		w.Crds.FormatA1(),
		w.MaxVol,
		w.WContents,
		w.Rvol,
		w.WShape,
		WellBottomNames[w.Bottom],
		w.GetSize().X,
		w.GetSize().Y,
		w.GetSize().Z,
		w.Bottomh,
		w.Extra,
		w.Plate,
	)
}

func (w *LHWell) Protected() bool {
	if w.Extra == nil {
		return false
	}

	p, ok := w.Extra["protected"]

	if !ok || !(p.(bool)) {
		return false
	}

	return true
}

func (w *LHWell) Protect() {
	if w.Extra == nil {
		w.Extra = make(map[string]interface{}, 3)
	}

	w.Extra["protected"] = true
}

func (w *LHWell) UnProtect() {
	if w.Extra == nil {
		w.Extra = make(map[string]interface{}, 3)
	}
	w.Extra["protected"] = false
}

func (w *LHWell) Contents() *LHComponent {
	// be careful
	if w == nil {
		logger.Debug("CONTENTS OF NIL WELL REQUESTED")
		return NewLHComponent()
	}
	if w.WContents == nil {
		return NewLHComponent()
	}

	return w.WContents
}

func (w *LHWell) Currvol() float64 {
	return w.Contents().Vol
}

func (w *LHWell) CurrVolume() wunit.Volume {
	return w.Contents().Volume()
}

func (w *LHWell) MaxVolume() wunit.Volume {
	return wunit.NewVolume(w.MaxVol, "ul")
}
func (w *LHWell) Add(c *LHComponent) error {
	//wasEmpty := w.Empty()
	mv := wunit.NewVolume(w.MaxVol, "ul")
	cv := wunit.NewVolume(c.Vol, "ul")
	wv := w.CurrentVolume()
	cv.Add(wv)
	if cv.GreaterThan(mv) {
		// could make this fatal but we don't track state well enough
		// for that to be worthwhile
		logger.Debug("WARNING: OVERFULL WELL AT ", w.Crds.FormatA1())
	}

	w.Contents().Mix(c)

	//if wasEmpty {
	// get rid of junk ID
	//	logger.Track(fmt.Sprintf("MIX REPLACED WELL CONTENTS ID WAS %s NOW %s", w.WContents.ID, c.ID))
	//w.WContents.ID = c.ID
	//}
	if cv.GreaterThan(mv) {
		return fmt.Errorf("Overfull well \"%s\", contains %s but maximum volume is only %s", w.GetName(), cv, mv)
	}
	return nil
}

func (w *LHWell) Remove(v wunit.Volume) (*LHComponent, error) {
	// if the volume is too high we complain

	if v.GreaterThan(w.CurrentVolume()) {
		//pid := "nil"
		//if p, ok := w.Plate.(*LHPlate); ok {
		//	pid = p.ID
		//}
		//logger.Debug("You ask too much: ", w.Crds.FormatA1(), " ", v.ToString(), " I only have: ", w.CurrentVolume().ToString(), " PLATEID: ", pid)
		return nil, fmt.Errorf("Requested %s from well \"%s\" which only contains %s", v, w.GetName(), w.CurrentVolume())
	}

	ret := w.Contents().Dup()
	ret.Vol = v.ConvertToString("ul")

	w.Contents().Remove(v)
	return ret, nil
}

func (w *LHWell) WorkingVolume() wunit.Volume {
	v := wunit.NewVolume(w.Currvol(), "ul")
	v2 := wunit.NewVolume(w.Rvol, "ul")
	v.Subtract(v2)
	return v
}

func (w *LHWell) ResidualVolume() wunit.Volume {
	v := wunit.NewVolume(w.Rvol, "ul")
	return v
}

func (w *LHWell) CurrentVolume() wunit.Volume {
	return w.Contents().Volume()
}

//@implement Location

func (lhw *LHWell) Location_ID() string {
	return lhw.ID
}

func (lhw *LHWell) Location_Name() string {
	return NameOf(lhw.Plate)
}

func (lhw *LHWell) Shape() *Shape {
	if lhw.WShape == nil {
		// return the non-shape
		return NewNilShape()
	}
	return lhw.WShape
}

// @implement Well
// @deprecate Well

func (w *LHWell) ContainerType() string {
	return TypeOf(w.Plate)
}

func (w *LHWell) Clear() {
	w.WContents = NewLHComponent()
	//death if this well is actually in a tipwaste
	w.WContents.Loc = w.Plate.(*LHPlate).ID + ":" + w.Crds.FormatA1()
}

func (w *LHWell) Empty() bool {
	if w.Currvol() <= 0.000001 {
		return true
	} else {
		return false
	}
}

// copy of instance
func (lhw *LHWell) Dup() *LHWell {
	ret := LHWell{
		GetUUID(),                //ID        string
		lhw.Inst,                 //Inst      string
		lhw.Crds,                 //Crds      WellCoords
		lhw.MaxVol,               //MaxVol    float64
		lhw.WContents.Dup(),      //WContents *LHComponent
		lhw.Rvol,                 //Rvol      float64
		lhw.WShape.Dup(),         //WShape    *Shape
		lhw.Bottom,               //Bottom    WellBottomType
		lhw.Bounds,               //Bounds    BBox
		lhw.Bottomh,              //Bottomh   float64
		map[string]interface{}{}, //Extra     map[string]interface{}
		lhw.Plate,                //Plate     LHObject `gotopb:"-" json:"-"`
	}

	for k, v := range lhw.Extra {
		ret.Extra[k] = v
	}

	return &ret
}

// copy of type
func (lhw *LHWell) CDup() *LHWell {
	cp := NewLHWell(lhw.Plate, lhw.Crds, "ul", lhw.MaxVol, lhw.Rvol, lhw.Shape().Dup(), lhw.Bottom, lhw.GetSize().X, lhw.GetSize().Y, lhw.GetSize().Z, lhw.Bottomh, "mm")
	for k, v := range lhw.Extra {
		cp.Extra[k] = v
	}

	return cp
}
func (lhw *LHWell) DupKeepIDs() *LHWell {
	cp := NewLHWell(lhw.Plate, lhw.Crds, "ul", lhw.MaxVol, lhw.Rvol, lhw.Shape().Dup(), lhw.Bottom, lhw.GetSize().X, lhw.GetSize().Y, lhw.GetSize().Z, lhw.Bottomh, "mm")

	for k, v := range lhw.Extra {
		cp.Extra[k] = v
	}

	// Dup here doesn't change ID
	cp.WContents = lhw.Contents().Dup()

	cp.ID = lhw.ID

	return cp
}

func (lhw *LHWell) CalculateMaxCrossSectionArea() (ca wunit.Area, err error) {

	ca, err = lhw.Shape().MaxCrossSectionalArea()

	return
}

func (lhw *LHWell) AreaForVolume() wunit.Area {

	ret := wunit.NewArea(0.0, "m^2")

	vf := lhw.GetAfVFunc()

	if vf == nil {
		ret, _ := lhw.CalculateMaxCrossSectionArea()
		return ret
	} else {
		vol := lhw.WContents.Volume()
		r := vf.F(vol.ConvertToString("ul"))
		ret = wunit.NewArea(r, "mm^2")
	}

	return ret
}

func (lhw *LHWell) HeightForVolume() wunit.Length {
	ret := wunit.NewLength(0.0, "m")

	return ret
}

func (lhw *LHWell) SetAfVFunc(f string) {
	lhw.Extra["afvfunc"] = f
}

func (lhw *LHWell) GetAfVFunc() wutil.Func1Prm {
	f, ok := lhw.Extra["afvfunc"]

	if !ok {
		return nil
	} else {
		x, err := wutil.UnmarshalFunc([]byte(f.(string)))
		if err != nil {
			logger.Fatal(fmt.Sprintf("Can't unmarshal function, error: %s", err.Error))
		}
		return x
	}
	return nil
}

func (lhw *LHWell) CalculateMaxVolume() (vol wunit.Volume, err error) {

	if lhw.Bottom == FlatWellBottom { // flat
		vol, err = lhw.Shape().Volume()
	} /*else if lhw.Bottom == UWellBottom { // round
		vol, err = lhw.Shape().Volume()
		// + additional calculation
	} else if lhw.Bottom == VWellBottom { // Pointed / v-shaped /pyramid
		vol, err = lhw.Shape().Volume()
		// + additional calculation
	}
	*/
	return
}

// make a new well structure
func NewLHWell(plate LHObject, crds WellCoords, vunit string, vol, rvol float64, shape *Shape, bott WellBottomType, xdim, ydim, zdim, bottomh float64, dunit string) *LHWell {
	var well LHWell

	well.Plate = plate
	well.WContents = NewLHComponent()
	//well.ID = "well-" + GetUUID()
	well.ID = GetUUID()
	well.Crds = crds
	well.MaxVol = wunit.NewVolume(vol, vunit).ConvertToString("ul")
	well.Rvol = wunit.NewVolume(rvol, vunit).ConvertToString("ul")
	well.WShape = shape.Dup()
	well.Bottom = bott
	well.Bounds = BBox{Coordinates{}, Coordinates{
		wunit.NewLength(xdim, dunit).ConvertToString("mm"),
		wunit.NewLength(ydim, dunit).ConvertToString("mm"),
		wunit.NewLength(zdim, dunit).ConvertToString("mm"),
	}}
	well.Bottomh = wunit.NewLength(bottomh, dunit).ConvertToString("mm")
	well.Extra = make(map[string]interface{})
	return &well
}

// this function tries to find somewhere to put something... it was written before
// i had an iterator. fml
func Get_Next_Well(plate *LHPlate, component *LHComponent, curwell *LHWell) (*LHWell, bool) {
	vol := component.Vol

	it := NewOneTimeColumnWiseIterator(plate)

	if curwell != nil {
		// quick check to see if we have room
		vol_left := get_vol_left(curwell)

		if vol_left >= vol {
			// fine we can just return this one
			return curwell, true
		}

		startcoords := curwell.Crds
		it.SetStartTo(startcoords)
		it.Rewind()
		it.Next()
	}

	var new_well *LHWell

	for wc := it.Curr(); it.Valid(); wc = it.Next() {

		crds := wc.FormatA1()

		new_well = plate.Wellcoords[crds]

		if new_well.Empty() {
			break
		}
		cnts := new_well.Contents()

		cont := cnts.Name()
		if cont != component.Name() {
			continue
		}

		vol_left := get_vol_left(new_well)

		if vol < vol_left {
			break
		}
	}

	if new_well == nil {
		return nil, false
	}

	return new_well, true
}

//XXX sloboda? This makes no sense now; need to revise
func get_vol_left(well *LHWell) float64 {
	//cnts := well.WContents
	// this is very odd... I can see how this works as a heuristic
	// but it doesn't make much sense to me
	carry_vol := 10.0 // microlitres
	//	total_carry_vol := float64(len(cnts)) * carry_vol
	total_carry_vol := carry_vol // yeah right
	Currvol := well.Currvol
	rvol := well.Rvol
	vol := well.MaxVol
	return vol - (Currvol() + total_carry_vol + rvol)
}

func (well *LHWell) DeclareTemporary() {
	if well != nil {

		if well.Extra == nil {
			well.Extra = make(map[string]interface{})
		}

		well.Extra["temporary"] = true
	} else {
		logger.Debug("Warning: Attempt to access nil well in DeclareTemporary()")
	}
}

func (well *LHWell) DeclareNotTemporary() {
	if well != nil {
		if well.Extra == nil {
			well.Extra = make(map[string]interface{})
		}
		well.Extra["temporary"] = false
	} else {
		logger.Debug("Warning: Attempt to access nil well in DeclareTemporary()")
	}
}

func (well *LHWell) IsTemporary() bool {
	if well != nil {
		if well.Extra == nil {
			return false
		}

		t, ok := well.Extra["temporary"]

		if !ok || !t.(bool) {
			return false
		}
		return true
	} else {
		logger.Debug("Warning: Attempt to access nil well in DeclareTemporary()")
	}
	return false
}

func (well *LHWell) DeclareAutoallocated() {
	if well != nil {

		if well.Extra == nil {
			well.Extra = make(map[string]interface{})
		}

		well.Extra["autoallocated"] = true
	} else {
		logger.Debug("Warning: Attempt to access nil well in DeclareAutoallocated()")
	}
}

func (well *LHWell) DeclareNotAutoallocated() {
	if well != nil {
		if well.Extra == nil {
			well.Extra = make(map[string]interface{})
		}
		well.Extra["autoallocated"] = false
	} else {
		logger.Debug("Warning: Attempt to access nil well in DeclareNotAutoallocated()")
	}
}

func (well *LHWell) IsAutoallocated() bool {
	if well != nil {
		if well.Extra == nil {
			return false
		}

		t, ok := well.Extra["autoallocated"]

		if !ok || !t.(bool) {
			return false
		}
		return true
	} else {
		logger.Debug("Warning: Attempt to access nil well in IsAutoallocated()")
	}
	return false
}

func (well *LHWell) Evaporate(time time.Duration, env Environment) VolumeCorrection {
	var ret VolumeCorrection

	// don't let this happen
	if well == nil {
		return ret
	}

	if well.Empty() {
		return ret
	}

	// we need to use the evaporation calculator
	// we should likely decorate wells since we have different capabilities
	// for different well types

	vol := eng.EvaporationVolume(env.Temperature, "water", env.Humidity, time.Seconds(), env.MeanAirFlowVelocity, well.AreaForVolume(), env.Pressure)

	r, _ := well.Remove(vol)

	if r == nil {
		well.WContents.Vol = 0.0
	}

	ret.Type = "Evaporation"
	ret.Volume = vol.Dup()
	ret.Location = well.WContents.Loc

	return ret
}

func (w *LHWell) ResetPlateID(newID string) {
	ltx := strings.Split(w.WContents.Loc, ":")
	w.WContents.Loc = newID + ":" + ltx[1]
	//w.Plateid = newID
}

func (w *LHWell) XDim() float64 {
	return w.Bounds.GetSize().X
}

func (w *LHWell) YDim() float64 {
	return w.Bounds.GetSize().Y
}
func (w *LHWell) ZDim() float64 {
	return w.Bounds.GetSize().Z
}

func (w *LHWell) IsUserAllocated() bool {
	if w.Extra == nil {
		return false
	}

	ua, ok := w.Extra["UserAllocated"].(bool)

	if !ok {
		return false
	}

	return ua
}

func (w *LHWell) SetUserAllocated() {
	if w.Extra == nil {
		w.Extra = make(map[string]interface{})
	}
	w.Extra["UserAllocated"] = true
}

func (w *LHWell) ClearUserAllocated() {
	if w.Extra == nil {
		w.Extra = make(map[string]interface{})
	}
	w.Extra["UserAllocated"] = false
}
