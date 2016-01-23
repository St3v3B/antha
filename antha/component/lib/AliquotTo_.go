// Variant of Aliquot where the low level MixTo command is used to pipette by
// row
package lib

import (
	"github.com/antha-lang/antha/antha/anthalib/mixer"
	"github.com/antha-lang/antha/antha/anthalib/wtype"
	"github.com/antha-lang/antha/antha/anthalib/wunit"
	"github.com/antha-lang/antha/antha/anthalib/wutil"
	"github.com/antha-lang/antha/bvendor/golang.org/x/net/context"
	"github.com/antha-lang/antha/execute"
	"github.com/antha-lang/antha/inject"
	"strconv"
)

// Input parameters for this protocol (data)

// Data which is returned from this protocol, and data types

// Physical Inputs to this protocol with types

// Physical outputs from this protocol with types

func _AliquotToRequirements() {

}

// Conditions to run on startup
func _AliquotToSetup(_ctx context.Context, _input *AliquotToInput) {

}

// The core process for this protocol, with the steps to be performed
// for every input
func _AliquotToSteps(_ctx context.Context, _input *AliquotToInput, _output *AliquotToOutput) {

	number := _input.SolutionVolume.SIValue() / _input.VolumePerAliquot.SIValue()
	possiblenumberofAliquots, _ := wutil.RoundDown(number)
	if possiblenumberofAliquots < _input.NumberofAliquots {
		panic("Not enough solution for this many aliquots")
	}

	aliquots := make([]*wtype.LHSolution, 0)

	// work out well coordinates for any plate
	wellpositionarray := make([]string, 0)

	//alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphabet := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
		"K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X",
		"Y", "Z", "AA", "BB", "CC", "DD", "EE", "FF"}
	//k := 0
	for j := 0; j < _input.OutPlate.WlsY; j++ {
		for i := 0; i < _input.OutPlate.WlsX; i++ { //countingfrom1iswhatmakesushuman := j + 1
			//k = k + 1
			wellposition := string(alphabet[j]) + strconv.Itoa(i+1)
			//fmt.Println(wellposition, k)
			wellpositionarray = append(wellpositionarray, wellposition)
		}

	}

	for k := 0; k < _input.NumberofAliquots; k++ {
		if _input.Solution.Type == "dna" {
			_input.Solution.Type = "DoNotMix"
		}
		aliquotSample := mixer.Sample(_input.Solution, _input.VolumePerAliquot)
		aliquot := execute.MixTo(_ctx, _input.OutPlate, wellpositionarray[k], aliquotSample)
		aliquots = append(aliquots, aliquot)
	}
	_output.Aliquots = aliquots
}

// Run after controls and a steps block are completed to
// post process any data and provide downstream results
func _AliquotToAnalysis(_ctx context.Context, _input *AliquotToInput, _output *AliquotToOutput) {
}

// A block of tests to perform to validate that the sample was processed correctly
// Optionally, destructive tests can be performed to validate results on a
// dipstick basis
func _AliquotToValidation(_ctx context.Context, _input *AliquotToInput, _output *AliquotToOutput) {

}
func _AliquotToRun(_ctx context.Context, input *AliquotToInput) *AliquotToOutput {
	output := &AliquotToOutput{}
	_AliquotToSetup(_ctx, input)
	_AliquotToSteps(_ctx, input, output)
	_AliquotToAnalysis(_ctx, input, output)
	_AliquotToValidation(_ctx, input, output)
	return output
}

func AliquotToRunSteps(_ctx context.Context, input *AliquotToInput) *AliquotToSOutput {
	soutput := &AliquotToSOutput{}
	output := _AliquotToRun(_ctx, input)
	if err := inject.AssignSome(output, &soutput.Data); err != nil {
		panic(err)
	}
	if err := inject.AssignSome(output, &soutput.Outputs); err != nil {
		panic(err)
	}
	return soutput
}

func AliquotToNew() interface{} {
	return &AliquotToElement{
		inject.CheckedRunner{
			RunFunc: func(_ctx context.Context, value inject.Value) (inject.Value, error) {
				input := &AliquotToInput{}
				if err := inject.Assign(value, input); err != nil {
					return nil, err
				}
				output := _AliquotToRun(_ctx, input)
				return inject.MakeValue(output), nil
			},
			In:  &AliquotToInput{},
			Out: &AliquotToOutput{},
		},
	}
}

var (
	_ = execute.MixInto
	_ = wunit.Make_units
)

type AliquotToElement struct {
	inject.CheckedRunner
}

type AliquotToInput struct {
	InPlate          *wtype.LHPlate
	NumberofAliquots int
	OutPlate         *wtype.LHPlate
	Solution         *wtype.LHComponent
	SolutionVolume   wunit.Volume
	VolumePerAliquot wunit.Volume
}

type AliquotToOutput struct {
	Aliquots []*wtype.LHSolution
}

type AliquotToSOutput struct {
	Data struct {
	}
	Outputs struct {
		Aliquots []*wtype.LHSolution
	}
}

func init() {
	addComponent(Component{Name: "AliquotTo",
		Constructor: AliquotToNew,
		Desc: ComponentDesc{
			Desc: "Variant of Aliquot where the low level MixTo command is used to pipette by\nrow\n",
			Path: "antha/component/an/Liquid_handling/Aliquot/AliquotTo.an",
			Params: []ParamDesc{
				{Name: "InPlate", Desc: "", Kind: "Inputs"},
				{Name: "NumberofAliquots", Desc: "", Kind: "Parameters"},
				{Name: "OutPlate", Desc: "", Kind: "Inputs"},
				{Name: "Solution", Desc: "", Kind: "Inputs"},
				{Name: "SolutionVolume", Desc: "", Kind: "Parameters"},
				{Name: "VolumePerAliquot", Desc: "", Kind: "Parameters"},
				{Name: "Aliquots", Desc: "", Kind: "Outputs"},
			},
		},
	})
}