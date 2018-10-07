// Copyright (C) 2018 spdf
// 
// This file is part of Elektron.
// 
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
//  any later version.
// 
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
// 

// TODO: Clean this up and use Mesos Attributes instead.
package constants

var Hosts = make(map[string]struct{})

/*
 Classification of the nodes in the cluster based on their Thermal Design Power (TDP).
*/
var PowerClasses = make(map[string]map[string]struct{})

// Threshold below which a host should be capped
var LowerCapLimit = 12.5
