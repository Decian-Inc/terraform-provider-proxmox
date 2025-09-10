package disk

import (
	"context"
	"strings"
	
	pveAPI "github.com/Telmate/proxmox-api-go/proxmox"
)

// PreserveCloudInitDrive checks if a cloud-init drive exists in the current VM configuration
// and preserves it by not marking it for deletion during updates.
// This handles two scenarios:
// 1. Cloud-init drives inherited from templates during cloning
// 2. Cloud-init drives automatically created by Proxmox when cloud-init parameters are set
func PreserveCloudInitDrive(ctx context.Context, client *pveAPI.Client, vmr *pveAPI.VmRef, storages *pveAPI.QemuStorages, hasCloudInitParams bool) error {
	// Get current VM configuration to check for existing cloud-init drives
	config, err := client.GetVmConfig(ctx, vmr)
	if err != nil {
		return err
	}

	// Check IDE slots for cloud-init drives
	if storages.Ide != nil {
		checkAndPreserveCloudInit(config, "ide0", &storages.Ide.Disk_0)
		checkAndPreserveCloudInit(config, "ide1", &storages.Ide.Disk_1)
		checkAndPreserveCloudInit(config, "ide2", &storages.Ide.Disk_2)
		checkAndPreserveCloudInit(config, "ide3", &storages.Ide.Disk_3)
	}

	// Check SATA slots for cloud-init drives
	if storages.Sata != nil {
		checkAndPreserveCloudInitSata(config, "sata0", &storages.Sata.Disk_0)
		checkAndPreserveCloudInitSata(config, "sata1", &storages.Sata.Disk_1)
		checkAndPreserveCloudInitSata(config, "sata2", &storages.Sata.Disk_2)
		checkAndPreserveCloudInitSata(config, "sata3", &storages.Sata.Disk_3)
		checkAndPreserveCloudInitSata(config, "sata4", &storages.Sata.Disk_4)
		checkAndPreserveCloudInitSata(config, "sata5", &storages.Sata.Disk_5)
	}

	// Check SCSI slots for cloud-init drives (less common but possible)
	if storages.Scsi != nil {
		checkAndPreserveCloudInitScsi(config, "scsi0", &storages.Scsi.Disk_0)
		checkAndPreserveCloudInitScsi(config, "scsi1", &storages.Scsi.Disk_1)
		checkAndPreserveCloudInitScsi(config, "scsi2", &storages.Scsi.Disk_2)
		checkAndPreserveCloudInitScsi(config, "scsi3", &storages.Scsi.Disk_3)
		// Add more SCSI slots if needed
	}
	
	// Also check if cloud-init parameters are present in the Terraform config
	// If they are, we should preserve any cloud-init drive that Proxmox creates
	if hasCloudInitParams {
		// Proxmox typically creates cloud-init drives on ide2 or ide3
		// Make sure we don't delete them if they exist
		preserveIfEmpty(config, "ide2", storages.Ide, 2)
		preserveIfEmpty(config, "ide3", storages.Ide, 3)
	}

	return nil
}

// checkAndPreserveCloudInit checks if a disk slot contains a cloud-init drive
// and if so, preserves it by not marking it for deletion
func checkAndPreserveCloudInit(config map[string]interface{}, diskSlot string, storage **pveAPI.QemuIdeStorage) {
	if storage == nil || *storage == nil {
		return
	}

	// Check if this disk slot exists in current configuration
	if diskConfig, exists := config[diskSlot]; exists {
		// Check if it's a cloud-init drive
		if diskStr, ok := diskConfig.(string); ok {
			// Cloud-init drives typically contain "cloudinit" in their configuration
			// Format is usually: storage:cloudinit or storage:vm-xxx-cloudinit
			if strings.Contains(diskStr, "cloudinit") {
				// Don't delete this disk slot - set it to nil to preserve it
				*storage = nil
			}
		}
	}
}

// Similar functions for SATA and SCSI storage types
func checkAndPreserveCloudInitSata(config map[string]interface{}, diskSlot string, storage **pveAPI.QemuSataStorage) {
	if storage == nil || *storage == nil {
		return
	}

	if diskConfig, exists := config[diskSlot]; exists {
		if diskStr, ok := diskConfig.(string); ok {
			if strings.Contains(diskStr, "cloudinit") {
				*storage = nil
			}
		}
	}
}

func checkAndPreserveCloudInitScsi(config map[string]interface{}, diskSlot string, storage **pveAPI.QemuScsiStorage) {
	if storage == nil || *storage == nil {
		return
	}

	if diskConfig, exists := config[diskSlot]; exists {
		if diskStr, ok := diskConfig.(string); ok {
			if strings.Contains(diskStr, "cloudinit") {
				*storage = nil
			}
		}
	}
}

// preserveIfEmpty preserves a disk slot if it exists in the config but would be deleted
// This is used when cloud-init parameters are present to preserve auto-created cloud-init drives
func preserveIfEmpty(config map[string]interface{}, diskSlot string, ide *pveAPI.QemuIdeDisks, slot int) {
	if ide == nil {
		return
	}
	
	// Check if this disk slot exists in current configuration
	if diskConfig, exists := config[diskSlot]; exists {
		if diskStr, ok := diskConfig.(string); ok {
			// If it contains cloudinit, preserve it
			if strings.Contains(diskStr, "cloudinit") {
				switch slot {
				case 0:
					if ide.Disk_0 != nil && ide.Disk_0.Delete {
						ide.Disk_0 = nil
					}
				case 1:
					if ide.Disk_1 != nil && ide.Disk_1.Delete {
						ide.Disk_1 = nil
					}
				case 2:
					if ide.Disk_2 != nil && ide.Disk_2.Delete {
						ide.Disk_2 = nil
					}
				case 3:
					if ide.Disk_3 != nil && ide.Disk_3.Delete {
						ide.Disk_3 = nil
					}
				}
			}
		}
	}
}
