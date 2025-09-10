package disk

import (
	"testing"
	
	pveAPI "github.com/Telmate/proxmox-api-go/proxmox"
)

func TestCheckAndPreserveCloudInit(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		diskSlot       string
		initialStorage *pveAPI.QemuIdeStorage
		expectedNil    bool
	}{
		{
			name: "Should preserve cloud-init drive",
			config: map[string]interface{}{
				"ide3": "local-lvm:vm-100-cloudinit",
			},
			diskSlot:       "ide3",
			initialStorage: &pveAPI.QemuIdeStorage{Delete: true},
			expectedNil:    true,
		},
		{
			name: "Should preserve cloud-init drive with different format",
			config: map[string]interface{}{
				"ide2": "local:cloudinit",
			},
			diskSlot:       "ide2",
			initialStorage: &pveAPI.QemuIdeStorage{Delete: true},
			expectedNil:    true,
		},
		{
			name: "Should not modify non-cloudinit drive",
			config: map[string]interface{}{
				"ide0": "local-lvm:vm-100-disk-0",
			},
			diskSlot:       "ide0",
			initialStorage: &pveAPI.QemuIdeStorage{Delete: true},
			expectedNil:    false,
		},
		{
			name: "Should handle missing disk slot",
			config: map[string]interface{}{
				"ide0": "local-lvm:vm-100-disk-0",
			},
			diskSlot:       "ide3",
			initialStorage: &pveAPI.QemuIdeStorage{Delete: true},
			expectedNil:    false,
		},
		{
			name:           "Should handle nil storage",
			config:         map[string]interface{}{},
			diskSlot:       "ide3",
			initialStorage: nil,
			expectedNil:    false, // nil remains nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := tt.initialStorage
			checkAndPreserveCloudInit(tt.config, tt.diskSlot, &storage)
			
			if tt.expectedNil && storage != nil {
				t.Errorf("Expected storage to be nil, but got %+v", storage)
			}
			if !tt.expectedNil && storage == nil {
				t.Errorf("Expected storage to not be nil")
			}
		})
	}
}

func TestPreserveCloudInitDrive_Integration(t *testing.T) {
	// Test with a mock configuration that has cloud-init on ide3
	config := map[string]interface{}{
		"ide3": "local-lvm:vm-100-cloudinit",
		"scsi0": "local-lvm:vm-100-disk-0",
	}

	storages := &pveAPI.QemuStorages{
		Ide: &pveAPI.QemuIdeDisks{
			Disk_0: &pveAPI.QemuIdeStorage{Delete: true},
			Disk_1: &pveAPI.QemuIdeStorage{Delete: true},
			Disk_2: &pveAPI.QemuIdeStorage{Delete: true},
			Disk_3: &pveAPI.QemuIdeStorage{Delete: true}, // This should be preserved
		},
		Scsi: &pveAPI.QemuScsiDisks{
			Disk_0: &pveAPI.QemuScsiStorage{Delete: true}, // This should remain as-is
		},
	}

	// Simulate the preservation logic
	checkAndPreserveCloudInit(config, "ide3", &storages.Ide.Disk_3)

	// Verify ide3 is preserved (set to nil)
	if storages.Ide.Disk_3 != nil {
		t.Errorf("Expected ide3 to be preserved (nil), but got %+v", storages.Ide.Disk_3)
	}

	// Verify other slots remain untouched
	if storages.Ide.Disk_0 == nil {
		t.Error("Expected ide0 to remain unchanged")
	}
	if storages.Scsi.Disk_0 == nil {
		t.Error("Expected scsi0 to remain unchanged")
	}
}
