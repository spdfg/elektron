// Copyright (C) 2018 spdf
// 
// This file is part of Elektron.
// 
// Elektron is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// Elektron is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with Elektron.  If not, see <http://www.gnu.org/licenses/>.
// 

package environment

// Environment Variables and that are used.
// These environment variables need to be set before launching elektron.

// Password for username="rapl" on the nodes in the cluster.
var RaplPassword = "RAPL_PSSWD"

// Location of the script that sets the powercap value for a host.
var RaplThrottleScriptLocation = "RAPL_PKG_THROTTLE_SCRIPT_LOCATION"
