// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deviations

import (
	"flag"

	"github.com/openconfig/ondatra"
)

// CPUMissingAncestor deviation set to true for devices where the CPU components
// do not map to a FRU parent component in the OC tree.
func CPUMissingAncestor(*ondatra.DUTDevice) bool {
	return *cpuMissingAncestor
}

// IntfRefConfigUnsupported deviation set to true for devices that do not support
// interface-ref configuration when applying features to interface.
func IntfRefConfigUnsupported(*ondatra.DUTDevice) bool {
	return *intfRefConfigUnsupported
}

// RequireRoutedSubinterface0 returns true if device needs to configure subinterface 0
// for non-zero sub-interfaces.
func RequireRoutedSubinterface0(*ondatra.DUTDevice) bool {
	return *requireRoutedSubinterface0
}

// GNOISwitchoverReasonMissingUserInitiated returns true for devices that don't
// report last-switchover-reason as USER_INITIATED for gNOI.SwitchControlProcessor.
func GNOISwitchoverReasonMissingUserInitiated(*ondatra.DUTDevice) bool {
	return *gnoiSwitchoverReasonMissingUserInitiated
}

// P4rtUnsetElectionIDPrimaryAllowed returns whether the device does not support unset election ID.
func P4rtUnsetElectionIDPrimaryAllowed(*ondatra.DUTDevice) bool {
	return *p4rtUnsetElectionIDPrimaryAllowed
}

// P4rtBackupArbitrationResponseCode returns whether the device does not support unset election ID.
func P4rtBackupArbitrationResponseCode(*ondatra.DUTDevice) bool {
	return *p4rtBackupArbitrationResponseCode
}

// BackupNHGRequiresVrfWithDecap returns true for devices that require
// IPOverIP Decapsulation for Backup NHG without interfaces.
func BackupNHGRequiresVrfWithDecap(*ondatra.DUTDevice) bool {
	return *backupNHGRequiresVrfWithDecap
}

var (
	cpuMissingAncestor                       = flag.Bool("deviation_cpu_missing_ancestor", false, "Set to true for devices where the CPU components do not map to a FRU parent component in the OC tree.")
	intfRefConfigUnsupported                 = flag.Bool("deviation_intf_ref_config_unsupported", false, "Set to true for devices that do not support interface-ref configuration when applying features to interface.")
	requireRoutedSubinterface0               = flag.Bool("deviation_require_routed_subinterface_0", false, "Set to true for a device that needs subinterface 0 to be routed for non-zero sub-interfaces")
	gnoiSwitchoverReasonMissingUserInitiated = flag.Bool("deviation_gnoi_switchover_reason_missing_user_initiated", false, "Set to true for devices that don't report last-switchover-reason as USER_INITIATED for gNOI.SwitchControlProcessor.")
	p4rtUnsetElectionIDPrimaryAllowed        = flag.Bool("deviation_p4rt_unsetelectionid_primary_allowed", false, "Device allows unset Election ID to be primary")
	p4rtBackupArbitrationResponseCode        = flag.Bool("deviation_bkup_arbitration_resp_code", false, "Device sets ALREADY_EXISTS status code for all backup client responses")
	backupNHGRequiresVrfWithDecap            = flag.Bool("deviation_backup_nhg_requires_vrf_with_decap", false, "Set to true for devices that require IPOverIP Decapsulation for Backup NHG without interfaces.")
)
