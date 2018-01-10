package wtype

// defines a tip waste
import "fmt"

// tip waste

type LHTipwaste struct {
	ID         string
	Type       string
	Mnfr       string
	Capacity   int
	Contents   int
	Height     float64
	WellXStart float64
	WellYStart float64
	WellZStart float64
	AsWell     *LHWell
}

func (tw LHTipwaste) SpaceLeft() int {
	return tw.Contents - tw.Capacity
}

func (te LHTipwaste) String() string {
	return fmt.Sprintf(
		`LHTipwaste {
	ID: %s,
	Type: %s,
	Mnfr: %s,
	Capacity: %d,
	Contents: %d,
	Height: %f,
	WellXStart: %f,
	WellYStart: %f,
	WellZStart: %f,
	AsWell: %p,
}
`,
		te.ID,
		te.Type,
		te.Mnfr,
		te.Capacity,
		te.Contents,
		te.Height,
		te.WellXStart,
		te.WellYStart,
		te.WellZStart,
		te.AsWell, //AsWell is printed as pointer to kepp things short
	)
}

func (tw *LHTipwaste) Dup() *LHTipwaste {
	tw2 := NewLHTipwaste(tw.Capacity, tw.Type, tw.Mnfr, tw.Height, tw.AsWell, tw.WellXStart, tw.WellYStart, tw.WellZStart)

	tw2.Contents = tw.Contents

	return tw2
}

func (tw *LHTipwaste) GetName() string {
	return tw.Type
}

func NewLHTipwaste(capacity int, typ, mfr string, height float64, w *LHWell, wellxstart, wellystart, wellzstart float64) *LHTipwaste {
	var lht LHTipwaste
	//	lht.ID = "tipwaste-" + GetUUID()
	lht.ID = GetUUID()
	lht.Type = typ
	lht.Mnfr = mfr
	lht.Capacity = capacity
	lht.Height = height
	lht.AsWell = w
	lht.WellXStart = wellxstart
	lht.WellYStart = wellystart
	lht.WellZStart = wellzstart
	return &lht
}

func (lht *LHTipwaste) Empty() {
	lht.Contents = 0
}

func (lht *LHTipwaste) Dispose(channels []*LHChannelParameter) bool {
	// this just checks numbers for now
	n := 0

	for _, c := range channels {
		if c != nil {
			n += 1
		}
	}

	if lht.Capacity-lht.Contents < n {
		return false
	}

	lht.Contents += n
	return true
}

/*
type SBSLabware interface {
	NumRows() int
	NumCols() int
	PlateHeight() float64
}
*/

// @implement SBSLabware

func (lht *LHTipwaste) NumRows() int {
	// this might change at some point...
	return 1
}

func (lht *LHTipwaste) NumCols() int {
	return 1
}

func (lht *LHTipwaste) PlateHeight() float64 {
	return lht.Height
}
