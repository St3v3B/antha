// liquidhandling/convertinstructions.go Part of the Antha language
// Copyright (C) 2015 The Antha authors. All rights reserved.
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
// contact license@antha-lang.org or write to the Antha team c/o
// Synthace Ltd. The London Bioscience Innovation Centre
// 2 Royal College St, London NW1 0NH UK

package liquidhandling

import (
	"context"
	"fmt"
	"github.com/antha-lang/antha/antha/anthalib/wtype"
	"github.com/antha-lang/antha/antha/anthalib/wunit"
)

func readableComponentArray(arr []*wtype.LHComponent) string {
	ret := ""

	for i, v := range arr {
		if v != nil {
			ret += fmt.Sprintf("%s:%-6.2f%s", v.CName, v.Vol, v.Vunit)
		} else {
			ret += "_nil_"
		}
		if i < len(arr)-1 {

			ret += ", "
		}
	}

	return ret
}

//
//	at this point (i.e. in a TransferBlock) the instructions have potentially been grouped into sets
//	with simultaneously servicable destinations - row or column-wise depending on the head
//	orientation chosen
//
//	The main goal here is to find sources in appropriate structure (rows or columns)
//	to allow them to be done simultaneously. This essentially follows a greedy strategy
//	in which the required components are aligned to the available sources to see how many
//	can be taken at once. There are some tricks involved to make this work with troughs
//	etc.
//

func ConvertInstructions(ctx context.Context, inssIn LHIVector, robot *LHProperties, carryvol wunit.Volume, channelprms *wtype.LHChannelParameter, multi int, legacyVolume bool, policy *wtype.LHPolicyRuleSet) (insOut []*TransferInstruction, robotOut *LHProperties, err error) {
	// again again again we dup the robot

	if multi != 1 {
		rbt := robot.DupKeepIDs()

		tfrs, err := convertInstructions(inssIn, rbt, carryvol, channelprms, multi, legacyVolume)

		if err != nil {
			return []*TransferInstruction{}, robot, err
		}

		// check if we actually generate any multichannel stuff here

		mcHere, err := hasMCB(ctx, tfrs, rbt, policy)

		if err != nil {
			return []*TransferInstruction{}, robot, err
		}

		// if there's no multichannel here require only single channeling
		// this is to make behaviour in regard of sources, atomic mixes
		// correct
		if !mcHere {
			multi = 1
		}
	}

	tfrs, err := convertInstructions(inssIn, robot, carryvol, channelprms, multi, legacyVolume)

	if err != nil {
		return []*TransferInstruction{}, robot, err
	}

	return tfrs, robot, nil
}

func hasMCB(ctx context.Context, tfrs []*TransferInstruction, rbt *LHProperties, policy *wtype.LHPolicyRuleSet) (bool, error) {
	hasMulti := false

	for _, tfr := range tfrs {
		instrx, err := tfr.Generate(ctx, policy, rbt)

		if err != nil {
			return false, err
		}

		for _, ins := range instrx {
			if ins.InstructionType() == MCB {
				hasMulti = true
				break
			}
		}

		if hasMulti {
			break
		}
	}

	return hasMulti, nil
}

func dupCmpAr(ins *wtype.LHInstruction) []*wtype.LHComponent {
	in := ins.Components
	r := make([]*wtype.LHComponent, len(in))

	for i := 0; i < len(in); i++ {
		r[i] = in[i].Dup()
		r[i].Loc = ins.PlateID + ":" + ins.Welladdress
	}

	return r
}

func convertInstructions(inssIn LHIVector, robot *LHProperties, carryvol wunit.Volume, channelprms *wtype.LHChannelParameter, multi int, legacyVolume bool) (insOut []*TransferInstruction, err error) {
	insOut = make([]*TransferInstruction, 0, 1)

	// TODO --> iterator?
	horiz := false

	var l int

	if multi == 1 {
		horiz = true
		l = len(inssIn)
	} else {
		horiz = false
		l = inssIn.MaxLen()
	}

	for i := 0; i < l; i++ {
		var inssToUse LHIVector
		var cmps []*wtype.LHComponent
		if horiz {
			if inssIn[i] == nil {
				continue
			}
			cmps = inssIn[i].ComponentsMoving()
			inssToUse = make(LHIVector, len(cmps))
			for j := 0; j < len(cmps); j++ {
				inssToUse[j] = inssIn[i]
			}
		} else {
			cmps = inssIn.CompsAt(i)
			inssToUse = inssIn
		}
		lenToMake := 0

		for _, c := range cmps {
			if c != nil {
				if c.CName == "" {
					panic("COMPONENTS MUST HAVE NAMES")
				}
				lenToMake += 1
			}
		}

		if lenToMake == 0 {
			// don't make empty transfers
			continue
		}

		orientation := wtype.LHVChannel
		independent := false

		if channelprms != nil {
			orientation = channelprms.Orientation
			independent = channelprms.Independent
		}

		parallelTransfers, err := robot.GetComponents(GetComponentsOptions{Cmps: cmps, Carryvol: carryvol, Ori: orientation, Multi: multi, Independent: independent, LegacyVolume: legacyVolume})

		if err != nil {
			return nil, err
		}

		for _, t := range parallelTransfers.Transfers {
			transfers, err := makeTransfers(t, cmps, robot, inssToUse, carryvol)

			if err != nil {
				return nil, err
			}

			insOut = append(insOut, transfers...)
		}

	}

	return insOut, nil
}

func makeTransfers(parallelTransfer ParallelTransfer, cmps []*wtype.LHComponent, robot *LHProperties, inssIn []*wtype.LHInstruction, carryvol wunit.Volume) ([]*TransferInstruction, error) {
	fromPlateIDs := parallelTransfer.PlateIDs
	fromWells := parallelTransfer.WellCoords
	vols := parallelTransfer.Vols

	insOut := make([]*TransferInstruction, 0, 1)
	// mt counts up the arrays got by GetComponents
	// each array refers to a transfer
	wh := make([]string, len(cmps))       //	what
	pf := make([]string, len(cmps))       //	position from
	pt := make([]string, len(cmps))       //	position to
	wf := make([]string, len(cmps))       //	well from
	wt := make([]string, len(cmps))       //	well to
	ptf := make([]string, len(cmps))      //	plate type from
	ptt := make([]string, len(cmps))      //	plate type to
	va := make([]wunit.Volume, len(cmps)) //	volume
	vf := make([]wunit.Volume, len(cmps)) //	volume in well from
	vt := make([]wunit.Volume, len(cmps)) //	volume in well to
	pfwx := make([]int, len(cmps))        //	plate from wells x
	pfwy := make([]int, len(cmps))        //	  "     "    "   y
	ptwx := make([]int, len(cmps))        //	  "    to    "   x
	ptwy := make([]int, len(cmps))        //	  "     "    "   y
	cnames := make([]string, len(cmps))   //        component names

	// ci counts up cmps

	for ci := 0; ci < len(cmps); ci++ {
		if len(fromPlateIDs) <= ci || fromPlateIDs[ci] == "" {
			continue
		}

		wh[ci] = cmps[ci].TypeName()

		// source plate position

		ppf, ok := robot.PlateIDLookup[fromPlateIDs[ci]]

		if !ok {
			return insOut, wtype.LHError(wtype.LH_ERR_DIRE, "Planning inconsistency: input plate ID not found on robot - please report this error to the authors")
		}

		pf[ci] = ppf

		// destination plate position

		ppt, ok := robot.PlateIDLookup[inssIn[ci].PlateID]

		if !ok {
			return insOut, wtype.LHError(wtype.LH_ERR_DIRE, "Planning inconsistency: destination plate ID not found on robot - please report this error to the authors")
		}

		pt[ci] = ppt

		// source well

		wf[ci] = fromWells[ci]

		wt[ci] = inssIn[ci].Welladdress

		// source plate type

		srcPlate, ok := robot.Plates[ppf]

		if !ok {
			return insOut, wtype.LHError(wtype.LH_ERR_DIRE, "Planning inconsistency: input plate ID not found on robot (#2) - please report this error to the authors")
		}

		ptf[ci] = srcPlate.Type

		// destination plate type

		dstPlate, ok := robot.Plates[ppt]

		if !ok {
			return insOut, wtype.LHError(wtype.LH_ERR_DIRE, "Planning inconsistency: destination plate ID not found on robot - please report this error to the authors")
		}

		ptt[ci] = dstPlate.Type

		// volume being moved

		va[ci] = vols[ci]

		// source well volume

		wellFrom, ok := srcPlate.Wellcoords[wf[ci]]

		if !ok {
			return insOut, wtype.LHError(wtype.LH_ERR_DIRE, "Planning inconsistency: source well not found on source plate - plate report this error to the authors")
		}

		vf[ci] = wellFrom.CurrVolume()

		// dest well volume

		wellTo, ok := dstPlate.Wellcoords[wt[ci]]

		if !ok {
			return insOut, wtype.LHError(wtype.LH_ERR_DIRE, "Planning inconsistency: dest well not found on dest plate - please report this error to the authors")
		}

		vt[ci] = wellTo.CurrVolume()

		// source plate dimensions

		pfwx[ci] = srcPlate.WellsX()
		pfwy[ci] = srcPlate.WellsY()

		// dest plate dimensions

		ptwx[ci] = dstPlate.WellsX()
		ptwy[ci] = dstPlate.WellsY()

		cnames[ci] = wellFrom.WContents.CName

		cmpFrom := wellFrom.Remove(va[ci])
		// silently remove the carry
		wellFrom.Remove(carryvol)

		if cmpFrom == nil {
			return insOut, wtype.LHError(wtype.LH_ERR_DIRE, "Planning inconsistency: src well does not contain sufficient volume - please report this error to the authors")
		}

		wellTo.Add(cmpFrom)

		// make sure the cmp loc is set

		wellTo.WContents.Loc = wellTo.Plateid + ":" + wellTo.Crds

		// make sure the wellTo gets the right ID (ultimately)
		cmpFrom.ReplaceDaughterID(wellTo.WContents.ID, inssIn[ci].Result.ID)
		wellTo.WContents.ID = inssIn[ci].Result.ID
		wellTo.WContents.DeclareInstance()
		//fmt.Println("ADDED :", cmpFrom.CName, " ", cmpFrom.Vol, " TO ", dstPlate.ID, " ", wt[ci])
	}

	//}

	tfr := NewTransferInstruction(wh, pf, pt, wf, wt, ptf, ptt, va, vf, vt, pfwx, pfwy, ptwx, ptwy, cnames)
	insOut = append(insOut, tfr)

	return insOut, nil
}
