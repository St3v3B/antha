// /anthalib/simulator/liquidhandling/simulator.go: Part of the Antha language
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
    "io/ioutil"
    "strings"
    "fmt"
	"github.com/antha-lang/antha/microArch/driver"
	"github.com/antha-lang/antha/microArch/driver/liquidhandling"
	"github.com/antha-lang/antha/microArch/simulator"
	"github.com/antha-lang/antha/antha/anthalib/wtype"
)

func summariseWell2Channel(well []string, channels []int) string {
    ret := make([]string, len(well))
    for i := range well {
        ret[i] = fmt.Sprintf("%s->channel%v", well[i], channels[i])
    }
    return strings.Join(ret, ", ")
}

// Simulate a liquid handler Driver
type VirtualLiquidHandler struct {
    simulator.ErrorReporter
    properties *liquidhandling.LHProperties 
    initialized bool
    finalized   bool
    log []string
}

//Create a new VirtualLiquidHandler which mimics an LHDriver
func NewVirtualLiquidHandler(props *liquidhandling.LHProperties) *VirtualLiquidHandler {
    var vlh VirtualLiquidHandler
    vlh.ErrorReporter = *simulator.NewErrorReporter()
    vlh.initialized = false
    vlh.finalized   = false

    vlh.properties = props.Dup()
    vlh.log = make([]string, 0)

    vlh.validateProperties()

    return &vlh
}

//Write to the log
func (self *VirtualLiquidHandler) LogLine(line string) {
    self.log = append(self.log, line)
}

//save the log
func (self *VirtualLiquidHandler) SaveLog(filename string) {
    ioutil.WriteFile(filename, []byte(strings.Join(self.log, "\n")), 0644)
}


// ------------------------------------------------------------------------------- Useful Utilities

func (self *VirtualLiquidHandler) validateProperties() {
    
    //check a property
    check_prop := func(l []string, name string) {
        //is empty
        if len(l) == 0 {
            self.AddWarningf("NewVirtualLiquidHandler", "No %s specified", name)
        }
        //all locations defined
        for _,loc := range l {
            if !self.locationIsKnown(loc) {
                self.AddWarningf("NewVirtualLiquidHandler", "Undefined location \"%s\" referenced in %s", loc, name)
            }
        }
    }

    check_prop(self.properties.Tip_preferences, "tip preferences")
    check_prop(self.properties.Input_preferences, "input preferences")
    check_prop(self.properties.Output_preferences, "output preferences")
    check_prop(self.properties.Tipwaste_preferences, "tipwaste preferences")
    check_prop(self.properties.Wash_preferences, "wash preferences")
    check_prop(self.properties.Waste_preferences, "waste preferences")
}

func (self *VirtualLiquidHandler) locationIsKnown(location string) bool {
    for loc := range self.properties.Layout {
        if loc == location {
            return true
        }
    }
    return false
}

func (self *VirtualLiquidHandler) checkReady(fnName string) {
    if !self.initialized {
        self.AddWarning(fnName, "Instruction given before Initialize")
    }
    if self.finalized {
        self.AddWarning(fnName, "Instruction given after Finalize")
    }
}

func (self *VirtualLiquidHandler) validateTipArgs(fnname string, 
                                        channels []int, head, multi int, 
                                        platetype, position, well []string) bool {
    //test length of channel, platetype, position, and well
    wrong_length := make([]string, 0)
    if len(channels) != multi {
        wrong_length = append(wrong_length, "channels")
    }
    if len(platetype) != multi {
        wrong_length = append(wrong_length, "platetype")
    }
    if len(position) != multi {
        wrong_length = append(wrong_length, "position")
    }
    if len(well) != multi {
        wrong_length = append(wrong_length, "well")
    }
    if len(wrong_length) > 0 {
        self.AddErrorf(fnname, "%s should be of length multi=%v",
                strings.Join(wrong_length, ", "),
                multi)
        return false
    }

    //check head exists
    if head < 0 || head >= len(self.properties.Heads) {
        self.AddErrorf(fnname, "Request for invalid Head %v", head)
        return false
    }

    adaptor := self.properties.Heads[head].Adaptor

    //check that channel values are valid and there are no duplicates
    encountered := map[int]bool{}
    for _, channel := range channels {
        if channel < 0 || channel >= adaptor.Params.Multi {
            self.AddErrorf(fnname, "Cannot load tip to channel %v of %v-channel adaptor", 
                            channel, adaptor.Params.Multi)
            return false
        }
        if encountered[channel] {
            self.AddErrorf(fnname, "Channel%v appears more than once", channel)
            return false
        } else {
            encountered[channel] = true
        }
    }

    return true
}

func elements_equal(slice []string) bool {
    for i := range slice {
        if slice[i] != slice[0] {
            return false
        }
    }
    return true
}

// ------------------------------------------------------------------------ ExtendedLHDriver

//Move command - used
func (self *VirtualLiquidHandler) Move(deckposition []string, wellcoords []string, reference []int, 
                                       offsetX, offsetY, offsetZ []float64, plate_type []string, 
                                       head int) driver.CommandStatus {
    self.checkReady("Move")
    self.LogLine(fmt.Sprintf(`Move(
    deckposition = %v,
    wellcoords = %v,
    reference = %v,
    offsetX,Y,Z = (%v, %v, %v),
    plate_type = %v,
    head = %v)`, deckposition, wellcoords, reference, offsetX, offsetY, offsetZ, plate_type, head))
    //Asserts:
    //deckposition exists - why is it a list?
    //wellcoords exists within the plate at deckposition
    //reference is in allowable range
    //offsetX, offsetY, offsetZ are within the well (but I guess they needn't be...)
    //plate_type matches the type of plate at deckposition
    //head is valid
    return driver.CommandStatus{true, driver.OK, "MOVE ACK"}
}

//Move raw - not yet implemented in compositerobotinstruction
func (self *VirtualLiquidHandler) MoveRaw(head int, x, y, z float64) driver.CommandStatus {
    self.checkReady("MoveRaw")
    self.LogLine(fmt.Sprintf(`MoveRaw(
    head = %v,
    offsetX,Y,Z = (%v, %v, %v))`, head, x,y,z))
    //Asserts:
    //head exists
    //x,y,x are within the machine
    return driver.CommandStatus{true, driver.OK, "MOVERAW ACK"}
}

//Aspirate - used
func (self *VirtualLiquidHandler) Aspirate(volume []float64, overstroke []bool, head int, multi int, 
                                           platetype []string, what []string, llf []bool) driver.CommandStatus {
    self.checkReady("Aspirate")
    self.LogLine(fmt.Sprintf(`Aspirate(
    volume = %v,
    overstroke = %v,
    head = %v,
    multi = %v,
    platetype = %v,
    what = %v,
    llf = %v)`, volume, overstroke, head, multi, platetype, what, llf))
    //volumes are equal if adapter isn't independent
    //tips are loaded in each adapter location the aspirates
    //volume is smaller that the tips' maximum capacity
    //the tip hasn't been used for a different liquid
    //head exists
    //multi matches number of tips loaded
    //platetype matches the plate at the location we moved to
    //what matches the expected liquid class
    //llf is the right size, cannot vary unless independent
    return driver.CommandStatus{true, driver.OK, "ASPIRATE ACK"}
}

//Dispense - used
func (self *VirtualLiquidHandler) Dispense(volume []float64, blowout []bool, head int, multi int, 
                                           platetype []string, what []string, llf []bool) driver.CommandStatus {
    self.checkReady("Dispense")
    self.LogLine(fmt.Sprintf(`Dispense(
    volume = %v,
    blowout = %v,
    head = %v,
    multi = %v,
    platetype = %v,
    what = %v,
    llf = %v)`, volume, blowout, head, multi, platetype, what, llf))
    //Volumes are equal if adapter isn't indepentent
    //Volumes are at most equal to the volume in the tip
    //blowout is the right length
    //head exists
    //multi is valid
    //platetype matches the type of plate that we're next to
    //what matches the liquid class that was aspirated
    //llf is the right size and follows independence constraint
    return driver.CommandStatus{true, driver.OK, "DISPENSE ACK"}
}

//LoadTips - used
func (self *VirtualLiquidHandler) LoadTips(channels []int, head, multi int, 
                                           platetype, position, well []string) driver.CommandStatus {
    self.checkReady("LoadTips")
    self.LogLine(fmt.Sprintf(`LoadTips(
    channels = %v,
    head = %v,
    multi = %v,
    platetype = %v,
    position = %v,
    well = %v)`, channels, head, multi, platetype, position, well))
    ret := driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    
    ok := self.validateTipArgs("LoadTips", channels, head, multi, platetype, position, well)
    if !ok {
        return ret
    }

    if !elements_equal(position) {
        self.AddError("LoadTips", "Cannot load tips from multiple locations")
        return ret
    }
    if !elements_equal(platetype) {
        self.AddError("LoadTips", "platetype should be equal")
        return ret
    }

    
    is_in_channels := map[int]bool{}
    for _,c := range channels {
        is_in_channels[c] = true
    }

    adaptor := self.properties.Heads[head].Adaptor
    tipbox, ok := self.properties.PlateLookup[self.properties.PosLookup[position[0]]].(*wtype.LHTipbox)
    if !ok {
        self.AddErrorf("LoadTips", "Cannot load tips from location \"%s\", no tipbox found", position[0])
        return driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    }
    if platetype[0] != tipbox.Type {
        self.AddErrorf("LoadTips", "Requested plate type \"%s\" but plate at %s is of type \"%s\"", platetype[0], position[0], tipbox.Type)
        return driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    }

    //check that the adapter is empty
    if adaptor.NTipsLoaded() != 0 {
        //because not having a ternary operator is so cool.
        stip := "tip"
        if adaptor.NTipsLoaded() > 1 {
            stip = "tips"
        }
        self.AddErrorf("LoadTips", "Cannot load tips while adaptor already contains %v %s", 
                adaptor.NTipsLoaded(), stip)
        return driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    }

    //check that well is valid
    wellcoords := make([]wtype.WellCoords, multi)
    wc_ok := true
    for i := range well {
        wellcoords[i] = wtype.MakeWellCoords(well[i])
        
        //check range
        if wellcoords[i].IsZero() {
            self.AddErrorf("LoadTips", "Couldn't parse well \"%s\"", well[i])
            wc_ok = false
            continue
        }
        if !tipbox.ContainsCoords(&wellcoords[i]) {
            self.AddErrorf("LoadTips", "Request for well %s, but tipbox size is [%vx%v]", well[i], tipbox.Ncols, tipbox.Nrows)
            wc_ok = false
            continue
        }
    }
    if !wc_ok {
        return driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    }

    //Check that the head origin is sane
    head_origin_ := make([]wtype.WellCoords, multi)
    for i,wc := range wellcoords {
        head_origin_[i] = wc
        if adaptor.Params.Orientation == wtype.LHVChannel {
            head_origin_[i].Y -= channels[i]
        } else if adaptor.Params.Orientation == wtype.LHHChannel {
            head_origin_[i].X -= channels[i]
        }
    }
    head_origin := head_origin_[0]
    origins_equal := true
    for _,o := range head_origin_ {
        origins_equal = origins_equal && head_origin.Equals(o)
    }
    if !origins_equal {
        self.AddErrorf("LoadTips", "Cannot load %s, tip spacing doesn't match channel spacing",
                summariseWell2Channel(well, channels))
        return driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    }
   
    //check for tip collisions on other channels
    if !adaptor.Params.Independent {
        collisions := []string{}
        for i := 0; i < adaptor.Params.Multi; i++ {
            pos := head_origin
            if adaptor.Params.Orientation == wtype.LHVChannel {
                pos.Y += i
            } else if adaptor.Params.Orientation == wtype.LHHChannel {
                pos.X += i
            }
            
            //if there's a tip at this location, and we don't intend to pick it up, and the adaptor is not independent
            if tipbox.HasTipAt(&pos) && !is_in_channels[i] {
               collisions = append(collisions, pos.FormatA1()) 
            }
        }
        if len(collisions) > 0 {
            stip := "tip"
            if len(collisions) > 1 {
                stip = "tips"
            }
            self.AddErrorf("LoadTips", "Cannot load %s due to %s at %s (Head%v is not independent)",
                    summariseWell2Channel(well, channels), stip, strings.Join(collisions, ","), head)
        }
    }


    //if we found an error, bail on the operation
    if self.HasError() {
        return driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    }
    
    //move the tips from the plate and add them to the adaptor
    var tip *wtype.LHTip
    for i := range well {
        
        tip = tipbox.Tips[wellcoords[i].X][wellcoords[i].Y]
        if tip == nil {
            self.AddErrorf("LoadTips", "Cannot load %s->channel%v as %s is empty", well[i], channels[i], well[i])
            continue
        }
        tipbox.Tips[wellcoords[i].X][wellcoords[i].Y] = nil
        //TODO: add the tip to the particular location in the adaptor
        adaptor.AddTip(channels[i], tip)
    }

    return driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
}

//UnloadTips - used
func (self *VirtualLiquidHandler) UnloadTips(channels []int, head, multi int, 
                                             platetype, position, well []string) driver.CommandStatus {
    self.checkReady("UnloadTips")
    self.LogLine(fmt.Sprintf(`UnloadTips(
    channels = %v,
    head = %v,
    multi = %v,
    platetype = %v,
    position = %v,
    well = %v)`, channels, head, multi, platetype, position, well))

    ret := driver.CommandStatus{true, driver.OK, "LOADTIPS ACK"}
    
    ok := self.validateTipArgs("UnloadTips", channels, head, multi, platetype, position, well)
    if !ok {
        return ret
    }

    if !elements_equal(position) {
        self.AddError("UnloadTips", "Cannot unload tips to multiple locations")
        return ret
    }
    if !elements_equal(platetype) {
        self.AddError("UnloadTips", "platetype should be equal")
        return ret
    }

    adaptor := self.properties.Heads[head].Adaptor
    tipwaste, ok := self.properties.PlateLookup[self.properties.PosLookup[position[0]]].(*wtype.LHTipwaste)
    if !ok {
        self.AddErrorf("UnloadTips", "Cannot unload tips at location \"%s\", no tipwaste found", position[0])
        return ret
    }
    if platetype[0] != tipwaste.Type {
        self.AddErrorf("UnloadTips", "Requested plate type \"%s\" but plate at %s is of type \"%s\"", platetype[0], position[0], tipwaste.Type)
        return ret
    }

    //if not independent, check we're unloading all tips
    if !adaptor.Params.Independent && len(channels) < adaptor.NTipsLoaded() {
        channel_str := "channel"
        if len(channels) > 1 {
            channel_str = "channels"
        }
        channels_str := []string{}
        for _,ch := range channels {
            channels_str = append(channels_str, fmt.Sprintf("%v", ch))
        }
        self.AddErrorf("UnloadTips", "Cannot unload tips from Head%v(%s %s) due to other tips on the adaptor (independent is false)",
            head,
            channel_str,
            strings.Join(channels_str, ","))
        return ret
    }

    //check there's a tip at every channel
    //TODO:
    if len(channels) > adaptor.NTipsLoaded() {
        if adaptor.NTipsLoaded() == 0 {
            self.AddErrorf("UnloadTips", "Cannot unload %v tips as no tips are loaded", len(channels))
        } else if adaptor.NTipsLoaded() == 1 {
            self.AddErrorf("UnloadTips", "Cannot unload %v tips as adaptor only has one tip loaded", len(channels))
        } else {
            self.AddErrorf("UnloadTips", "Cannot unload %v tips as adaptor only has %v tips loaded", len(channels), adaptor.NTipsLoaded())
        }
        return ret
    }

    //Actually unload the adaptor
    for _,ch := range channels {
        adaptor.RemoveTip(ch)
    }
    tipwaste.Contents += len(channels)

    return ret
}

//SetPipetteSpeed - used
func (self *VirtualLiquidHandler) SetPipetteSpeed(head, channel int, rate float64) driver.CommandStatus {
    self.checkReady("SetPipetteSpeed")
    self.LogLine(fmt.Sprintf(`SetPipetteSpeed(
    head = %v,
    channel = %v,
    rate = %v)`, head, channel, rate))
    //head exists
    //channel exists
    //speed is within allowable range
    return driver.CommandStatus{true, driver.OK, "SETPIPETTESPEED ACK"}
}

//SetDriveSpeed - used
func (self *VirtualLiquidHandler) SetDriveSpeed(drive string, rate float64) driver.CommandStatus {
    self.checkReady("SetDriveSpeed")
    self.LogLine(fmt.Sprintf(`SetDriveSpeed(
    drive = %v,
    rate = %v)`, drive, rate))
    //drive string?
    //rate is within allowable range (what is this?)
    return driver.CommandStatus{true, driver.OK, "SETDRIVESPEED ACK"}
}

//Stop - unused
func (self *VirtualLiquidHandler) Stop() driver.CommandStatus {
    panic("unimplemented")
}

//Go - unused
func (self *VirtualLiquidHandler) Go() driver.CommandStatus {
    self.checkReady("Go")
    panic("unimplemented")
}

//Initialize - used
func (self *VirtualLiquidHandler) Initialize() driver.CommandStatus {
    self.LogLine("Initialize()")
    if self.initialized {
        self.AddWarningf("Initialize", "Multiple calls to Initialize")
    } else {
        self.initialized = true
    }
    return driver.CommandStatus{true, driver.OK, "INITIALIZE ACK"}
}

//Finalize - used
func (self *VirtualLiquidHandler) Finalize() driver.CommandStatus {
    self.LogLine("Finalize()")
    //check that this is called last, no more calls
    if self.finalized {
        self.AddWarningf("Finalize", "Multiple calls to Finalize")
    } else {
        self.finalized = true
    }
    return driver.CommandStatus{true, driver.OK, "FINALIZE ACK"}
}

//Wait - used
func (self *VirtualLiquidHandler) Wait(time float64) driver.CommandStatus {
    self.checkReady("Wait")
    self.LogLine(fmt.Sprintf(`Wait(time = %v)`, time))
    //time is positive
    //maybe a warning if it's super-long
    return driver.CommandStatus{true, driver.OK, "WAIT ACK"}
}

//Mix - used
func (self *VirtualLiquidHandler) Mix(head int, volume []float64, platetype []string, cycles []int, 
                                      multi int, what []string, blowout []bool) driver.CommandStatus {
    self.checkReady("Mix")
    self.LogLine(fmt.Sprintf(`Mix(
    head = %v,
    volume = %v,
    platetype = %v,
    cycles = %v,
    multi = %v,
    what = %v,
    blowout = %v)`, head, volume, platetype, cycles, multi, what, blowout))
    //head exists
    //volume is lte volume in wells
    //platetype matches the plate we're over
    //muli is correct
    //what matches expected liquidclass
    //volume, platetype, what, blowout match independence constraint
    return driver.CommandStatus{true, driver.OK, "MIX ACK"}
}

//ResetPistons - used
func (self *VirtualLiquidHandler) ResetPistons(head, channel int) driver.CommandStatus {
    self.checkReady("ResetPistons")
    self.LogLine("ResetPistons()")
    //head exists
    //channel exists
    //what does this do again? probably need to make sure it gets called appropriately
    return driver.CommandStatus{true, driver.OK, "RESETPISTONS ACK"}
}

//AddPlateTo - used
func (self *VirtualLiquidHandler) AddPlateTo(position string, plate interface{}, name string) driver.CommandStatus {
    self.checkReady("AddPlateTo")
    self.LogLine(fmt.Sprintf(`AddPlateTo(
    position = %v,
    plate = %v,
    name = %v)`, position, plate, name))
    //check that the requested position exists
    if !self.locationIsKnown(position) {
        self.AddWarningf("AddPlateTo", "Adding plate \"%s\" to unknown location \"%s\"", name, position)
    }
    //check that the requested position is empty
    if self.properties.PosLookup[position] != "" {
        self.AddErrorf("AddPlateTo", "Cannot add plate \"%s\" to location \"%s\" which is already occupied by plate \"%s\"", 
                name, 
                position, 
                self.properties.PlateLookup[self.properties.PosLookup[position]].(wtype.Named).GetName())
    } else {
        //position can accept a plate of this type
        switch plate := plate.(type) {
        case *wtype.LHPlate:
            plate.PlateName = name
            self.properties.AddPlate(position, plate)
        case *wtype.LHTipbox:
            plate.Boxname = name
            self.properties.AddTipBoxTo(position, plate)
        case *wtype.LHTipwaste:
            plate.Type = name
            self.properties.AddTipWasteTo(position, plate)

        default:
            self.AddErrorf("AddPlateTo", "Cannot add plate \"%s\" of type %T to location \"%s\"", name, plate, position)
        }
    }
    return driver.CommandStatus{true, driver.OK, "ADDPLATETO ACK"}
}

//RemoveAllPlates - used
func (self *VirtualLiquidHandler) RemoveAllPlates() driver.CommandStatus {
    self.checkReady("RemoveAllPlates")
    self.LogLine("RemoveAllPlates()")
    //remove plates, no checks required.
    return driver.CommandStatus{true, driver.OK, "REMOVEALLPLATES ACK"}
}

//RemovePlateAt - unused
func (self *VirtualLiquidHandler) RemovePlateAt(position string) driver.CommandStatus {
    self.LogLine(fmt.Sprintf("RemovePlateAt(position = %v)", position))
    //plate exists at position
    return driver.CommandStatus{true, driver.OK, "REMOVEPLATEAT ACK"}
}

//SetPositionState - unused
func (self *VirtualLiquidHandler) SetPositionState(position string, state driver.PositionState) driver.CommandStatus {
    panic("unimplemented")
}

//GetCapabilites - used
func (self *VirtualLiquidHandler) GetCapabilities() (liquidhandling.LHProperties, driver.CommandStatus) {
    self.checkReady("GetCapabilities")
    self.LogLine("GetCapabilities()")
    //no checks required
    return *self.properties, driver.CommandStatus{true, driver.OK, ""} 
}

//GetCurrentPosition - unused
func (self *VirtualLiquidHandler) GetCurrentPosition(head int) (string, driver.CommandStatus) {
    panic("unimplemented")
}

//GetPositionState - unused
func (self *VirtualLiquidHandler) GetPositionState(position string) (string, driver.CommandStatus) {
    panic("unimplemented")
}

//GetHeadState - unused
func (self *VirtualLiquidHandler) GetHeadState(head int) (string, driver.CommandStatus) {
    panic("unimplemented")
}

//GetStatus - unused
func (self *VirtualLiquidHandler) GetStatus() (driver.Status, driver.CommandStatus) {
    panic("unimplemented")
}

//UpdateMetaData - used
func (self *VirtualLiquidHandler) UpdateMetaData(props *liquidhandling.LHProperties) driver.CommandStatus {
    self.checkReady("UpdateMetaData")
    self.LogLine("UpdateMetaData(props *LHProperties)")
    //check that the props and self.props are the same...
    return driver.CommandStatus{true, driver.OK, "UPDATEMETADATA ACK"}
}

//UnloadHead - unused
func (self *VirtualLiquidHandler) UnloadHead(param int) driver.CommandStatus {
    panic("unimplemented")
}

//LoadHead - unused
func (self *VirtualLiquidHandler) LoadHead(param int) driver.CommandStatus {
    panic("unimplemented")
}

//Lights On - not implemented in compositerobotinstruction
func (self *VirtualLiquidHandler) LightsOn() driver.CommandStatus {
    panic("unimplemented")
}

//Lights Off - notimplemented in compositerobotinstruction
func (self *VirtualLiquidHandler) LightsOff() driver.CommandStatus {
    panic("unimplemented")
}

//LoadAdaptor - notimplemented in CRI
func (self *VirtualLiquidHandler) LoadAdaptor(param int) driver.CommandStatus {
    panic("unimplemented")
}

//UnloadAdaptor - notimplemented in CRI
func (self *VirtualLiquidHandler) UnloadAdaptor(param int) driver.CommandStatus {
    panic("unimplemented")
}

//Open - notimplemented in CRI
func (self *VirtualLiquidHandler) Open() driver.CommandStatus {
    panic("unimplemented")
}

//Close - notimplement in CRI
func (self *VirtualLiquidHandler) Close() driver.CommandStatus {
    panic("unimplemented")
}

//Message - unused
func (self *VirtualLiquidHandler) Message(level int, title, text string, showcancel bool) driver.CommandStatus {
    panic("unimplemented")
}

//GetOutputFile - used, but not in instruction stream
func (self *VirtualLiquidHandler) GetOutputFile() (string, driver.CommandStatus) {
    //Probably won't get called on the simulator just yet...
    panic("unimplemented")
}


