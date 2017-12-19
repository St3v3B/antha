// antha/AnthaStandardLibrary/Packages/enzymes/Find.go: Part of the Antha language
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

// Utility package providing functions useful for searches
package search

import (
	"strings"
)

type Named interface {
	Name() string
}

// EqualName evaluates whether two arguments with a Name() method have equal names.
// If the IgnoreCase option is specified the strings will be compared ignoring case.
func EqualName(entry, target Named, options ...Option) bool {

	ignore := containsIgnoreCase(options...)

	if ignore {
		if equalFold(entry.Name(), target.Name()) {
			return true
		}
	} else {
		if strings.TrimSpace(entry.Name()) == strings.TrimSpace(target.Name()) {
			return true
		}
	}
	return false
}
