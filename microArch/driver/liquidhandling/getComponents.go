package liquidhandling

// func (lhp *LHProperties) GetComponents(cmps []*wtype.LHComponent, carryvol wunit.Volume, ori, multi int, independent, legacyVolume bool) (plateIDs, wellCoords [][]string, vols [][]wunit.Volume, err error)

import (
	"fmt"
	"github.com/antha-lang/antha/antha/anthalib/wtype"
	"github.com/antha-lang/antha/antha/anthalib/wunit"
)

type ComponentVolumeHash map[string]wunit.Volume

func (h ComponentVolumeHash) AllVolsPosOrZero() bool {
	for _, v := range h {
		if v.LessThan(wunit.ZeroVolume()) {
			return false
		}
	}
	return true
}

func (h ComponentVolumeHash) Dup() ComponentVolumeHash {
	r := make(ComponentVolumeHash, len(h))
	for k, v := range h {
		r[k] = v.Dup()
	}

	return r
}

type GetComponentsOptions struct {
	Cmps         wtype.ComponentVector
	Carryvol     wunit.Volume
	Ori          int
	Multi        int
	Independent  bool
	LegacyVolume bool
}

type ParallelTransfer struct {
	PlateIDs   []string
	WellCoords []string
	Vols       []wunit.Volume
}

type GetComponentsReply struct {
	Transfers []ParallelTransfer
}

func newReply() GetComponentsReply {
	return GetComponentsReply{Transfers: make([]ParallelTransfer, 0, 1)}
}

func areWeDoneYet(cmps wtype.ComponentVector) bool {
	for _, c := range cmps {
		if c != nil && c.Vol != 0 {
			return false
		}
	}

	return true
}

func matchToParallelTransfer(m wtype.Match) ParallelTransfer {
	return ParallelTransfer{PlateIDs: m.IDs, WellCoords: m.WCs, Vols: m.Vols}
}

// returns a vector iterator for a plate given the multichannel capabilites of the head (ori, multi)
func getPlateIterator(lhp *wtype.LHPlate, ori, multi int) wtype.VectorPlateIterator {
	if ori == wtype.LHVChannel {
		//it = NewColVectorIterator(lhp, multi)

		tpw := multi / lhp.WellsY()
		wpt := lhp.WellsY() / multi

		if tpw == 0 {
			tpw = 1
		}

		if wpt == 0 {
			wpt = 1
		}

		// fix for 6 row plates etc.
		if multi > lhp.WellsY() && tpw == 1 {
			multi = lhp.WellsY()
		}

		return wtype.NewTickingColVectorIterator(lhp, multi, tpw, wpt)
	} else {
		// needs same treatment as above
		return wtype.NewRowVectorIterator(lhp, multi)
	}
}

func (lhp *LHProperties) GetSourcesFor(cmps wtype.ComponentVector, ori, multi int, minPossibleVolume wunit.Volume) []wtype.ComponentVector {
	ret := make([]wtype.ComponentVector, 0, 1)

	for _, ipref := range lhp.OrderedMergedPlatePrefs() {
		p, ok := lhp.Plates[ipref]

		if ok {
			it := getPlateIterator(p, ori, multi)

			for wv := it.Curr(); it.Valid(); wv = it.Next() {
				// cmps needs duping here
				mycmps := p.GetVolumeFilteredContentVector(wv, cmps, minPossibleVolume) // dups components
				if mycmps.Empty() {
					continue
				}

				// mycmps has incorrect volumes, try correcting them here

				correct_volumes(mycmps)

				ret = append(ret, mycmps)
			}
		}
	}

	return ret
}

func correct_volumes(cmps wtype.ComponentVector) {
	nW := make(map[string]int)
	for _, c := range cmps {
		if c == nil {
			continue
		}

		_, ok := nW[c.Loc]
		if !ok {
			nW[c.Loc] = 0
		}
		nW[c.Loc] += 1
	}

	for _, c := range cmps {
		if c == nil {
			continue
		}

		c.Vol /= float64(nW[c.Loc])
	}
}

func cullZeroes(m map[string]wunit.Volume) map[string]wunit.Volume {
	r := make(map[string]wunit.Volume, len(m))

	for k, v := range m {
		if v.IsZero() {
			continue
		}

		r[k] = v
	}

	return r
}

func sourceVolumesOK(srcs []wtype.ComponentVector, dests wtype.ComponentVector) (bool, string) {
	collSrcs := sumSources(srcs)
	collDsts := dests.ToSumHash()
	collDsts = cullZeroes(collDsts)

	result := subHash(collSrcs, collDsts)

	if len(collSrcs) < len(collDsts) {
		return false, collateDifference(collDsts, collSrcs, result)
	}

	r := result.AllVolsPosOrZero()

	if r {
		return r, ""
	} else {
		return r, collateDifference(collDsts, collSrcs, result)
	}
}

func collateDifference(a, b, c map[string]wunit.Volume) string {
	s := ""

	for k, _ := range a {
		_, ok := b[k]

		if !ok {
			s += fmt.Sprintf("%s; ", k)
			continue
		}

		v := c[k]

		if v.LessThanFloat(0.0) {
			v.M(-1.0)
			s += fmt.Sprintf("%s - missing %s; ", k, v.ToString())
		}
	}

	return s
}

func subHash(h1, h2 ComponentVolumeHash) ComponentVolumeHash {
	r := h1.Dup()
	for k, v := range h2 {
		_, ok := r[k]

		if ok {
			r[k].Subtract(v)
		}
	}

	return r
}

func sumSources(cmpV []wtype.ComponentVector) ComponentVolumeHash {
	ret := make(ComponentVolumeHash, len(cmpV))
	for _, cV2 := range cmpV {
		for _, c := range cV2 {
			if c != nil && c.CName != "" {
				v, ok := ret[c.FullyQualifiedName()]
				if !ok {
					v = wunit.NewVolume(0.0, "ul")
					ret[c.FullyQualifiedName()] = v
				}
				v.Add(c.Volume())
			}
		}
	}

	return ret
}

func cmpVecsEqual(v1, v2 wtype.ComponentVector) bool {
	if len(v1) != len(v2) {
		return false
	}

	for i := 0; i < len(v1); i++ {
		if !cmpsEqual(v1[i], v2[i]) {
			return false
		}
	}

	return true
}

func cmpsEqual(c1, c2 *wtype.LHComponent) bool {
	return c1.ID == c2.ID && c1.Vol == c2.Vol
}

func (lhp *LHProperties) GetComponents(opt GetComponentsOptions) (GetComponentsReply, error) {
	rep := newReply()
	// build list of possible sources -- this is a list of ComponentVectors

	srcs := lhp.GetSourcesFor(opt.Cmps, opt.Ori, opt.Multi, lhp.MinPossibleVolume())

	// keep taking chunks until either we get everything or run out
	// optimization options apply here as parameters for the next level down

	currCmps := opt.Cmps.Dup()
	var lastCmps wtype.ComponentVector

	done := false

	for {
		done = areWeDoneYet(currCmps)
		if done {
			break
		}

		if ok, s := sourceVolumesOK(srcs, currCmps); !ok {
			return GetComponentsReply{}, fmt.Errorf("Insufficient source volumes for components %s", s)
		}

		if cmpVecsEqual(lastCmps, currCmps) {
			// if we are here we should be able to service the request but not
			// as-is...
			break
		}

		bestMatch := wtype.Match{Sc: -1.0}
		var bestSrc wtype.ComponentVector
		// srcs is chunked up to conform to what can be accessed by the LH
		for _, src := range srcs {
			if src.Empty() {
				continue
			}

			match, err := wtype.MatchComponents(currCmps, src, opt.Independent, false)

			if err != nil && err.Error() != wtype.NotFoundError {
				return rep, err
			}

			if match.Sc > bestMatch.Sc {
				bestMatch = match
				bestSrc = src
			}
		}

		if bestMatch.Sc == -1 {
			return rep, fmt.Errorf("Components %s %s -- try increasing source volumes, if this does not work or is not possible please report to the authors\n", currCmps.String(), wtype.NotFoundError)
		}

		// adjust finally to ensure we don't leave too little

		bestMatch = makeMatchSafe(currCmps, bestMatch, lhp.MinPossibleVolume())

		// update sources

		updateSources(bestSrc, bestMatch, opt.Carryvol, lhp.MinPossibleVolume())
		lastCmps = currCmps.Dup()
		updateDests(currCmps, bestMatch)
		rep.Transfers = append(rep.Transfers, matchToParallelTransfer(bestMatch))
	}

	return rep, nil
}

// this double-checks if we are using duplicated trough wells
func feasible(match wtype.Match, src wtype.ComponentVector, carry wunit.Volume) bool {
	// sum available volumes asked for and those available

	want := make(map[string]wunit.Volume)

	for i := 0; i < len(match.IDs); i++ {
		if match.M[i] == -1 {
			continue
		}
		if _, ok := want[match.IDs[i]+":"+match.WCs[i]]; !ok {
			want[match.IDs[i]+":"+match.WCs[i]] = wunit.NewVolume(0.0, "ul")
		}
		want[match.IDs[i]+":"+match.WCs[i]].Add(match.Vols[i])
		want[match.IDs[i]+":"+match.WCs[i]].Add(carry)
	}

	got := make(map[string]wunit.Volume)

	for i := 0; i < len(src); i++ {
		// if a component appears more than once in a location it's a fake duplicate
		got[src[i].Loc] = src[i].Volume()
	}

	compare := func(a, b map[string]wunit.Volume) bool {
		// true iff all volumes in a are <= their equivalents in b (undef == 0)
		for k, v1 := range a {
			v2, ok := b[k]
			if !ok {
				return false
			}

			if v2.LessThan(v1) {
				return false
			}
		}

		return true
	}

	return compare(want, got)
}

func updateSources(src wtype.ComponentVector, match wtype.Match, carryVol, minPossibleVolume wunit.Volume) wtype.ComponentVector {
	for i := 0; i < len(match.M); i++ {
		if match.M[i] != -1 {
			volSub := wunit.CopyVolume(match.Vols[i])
			volSub.Add(carryVol)
			src[match.M[i]].Vol -= volSub.ConvertToString(src[match.M[i]].Vunit)
		}
	}

	src.DeleteAllBelowVolume(minPossibleVolume)

	return src
}

func makeMatchSafe(dst wtype.ComponentVector, match wtype.Match, mpv wunit.Volume) wtype.Match {
	for i := 0; i < len(match.M); i++ {
		if match.M[i] != -1 {
			checkVol := dst[i].Vol

			checkVol -= match.Vols[i].ConvertToString(dst[i].Vunit)

			if checkVol > 0.0 && checkVol < mpv.ConvertToString(dst[i].Vunit) {
				mpv.Subtract(wunit.NewVolume(checkVol, dst[i].Vunit))
				match.Vols[i].Subtract(mpv)

				if match.Vols[i].LessThanFloat(0.0) {
					panic(fmt.Sprintf("Serious volume issue -- try a manual plate layout with some additional volume for %s", dst[i].CName))
				}
			}
		}
	}

	return match
}

func updateDests(dst wtype.ComponentVector, match wtype.Match) wtype.ComponentVector {
	for i := 0; i < len(match.M); i++ {
		if match.M[i] != -1 {
			dst[i].Vol -= match.Vols[i].ConvertToString(dst[i].Vunit)
			if dst[i].Vol < 0.0 {
				dst[i].Vol = 0.0
			}
		}
	}

	return dst
}
