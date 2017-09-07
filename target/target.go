// Package target provides the construction of a target machine from a
// collection of devices
package target

import (
	"context"
	"errors"

	"github.com/antha-lang/antha/ast"
)

var (
	errNoLh         = errors.New("no liquid handler found")
	errNoTarget     = errors.New("no target configuration found")
	errAlreadyAdded = errors.New("already added")
)

const (
	// DriverSelectorV1Name is the basic selector name for device plugins
	// (drivers)
	DriverSelectorV1Name = "antha.driver.v1.TypeReply.type"
)

// Well known device plugins (drivers) selectors
var (
	DriverSelectorV1Human = ast.NameValue{
		Name:  DriverSelectorV1Name,
		Value: "antha.human.v1.Human",
	}
	DriverSelectorV1ShakerIncubator = ast.NameValue{
		Name:  DriverSelectorV1Name,
		Value: "antha.shakerincubator.v1.ShakerIncubator",
	}
	DriverSelectorV1Mixer = ast.NameValue{
		Name:  DriverSelectorV1Name,
		Value: "antha.mixer.v1.Mixer",
	}
	DriverSelectorV1Prompter = ast.NameValue{
		Name:  DriverSelectorV1Name,
		Value: "antha.prompter.v1.Prompter",
	}
)

type targetKey int

const theTargetKey targetKey = 0

// GetTarget returns the current Target in context
func GetTarget(ctx context.Context) (*Target, error) {
	v, ok := ctx.Value(theTargetKey).(*Target)
	if !ok {
		return nil, errNoTarget
	}
	return v, nil
}

// WithTarget creates a context with the given Target
func WithTarget(parent context.Context, t *Target) context.Context {
	return context.WithValue(parent, theTargetKey, t)
}

// Target machine for execution.
type Target struct {
	devices []Device
}

// New creates a new target
func New() *Target {
	return &Target{}
}

func (a *Target) canCompile(d Device, reqs ...ast.Request) bool {
	for _, req := range reqs {
		if !d.CanCompile(req) {
			return false
		}
	}
	return true
}

// CanCompile returns the devices that can compile the given set of requests
func (a *Target) CanCompile(reqs ...ast.Request) (r []Device) {
	for _, d := range a.devices {
		if a.canCompile(d, reqs...) {
			r = append(r, d)
		}
	}
	return
}

// AddDevice adds a device to the target configuration
func (a *Target) AddDevice(d Device) error {
	a.devices = append(a.devices, d)
	return nil
}
