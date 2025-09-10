# Cloud-Init Drive Deletion Bug Fix

## Issue Summary
As reported in [GitHub Issue #901](https://github.com/Telmate/terraform-provider-proxmox/issues/901), cloud-init drives are being deleted after VM creation during move and resize operations. This occurs in two scenarios:

1. When cloning a template that has a cloud-init drive attached
2. When cloud-init parameters are specified in Terraform configuration (even if the template doesn't have a cloud-init drive)

## Root Cause Analysis

The issue occurs in the following sequence:

### Scenario 1: Template with Cloud-Init Drive
1. **VM Cloning**: When a VM is cloned from a template, it inherits all disks including the cloud-init drive (typically on ide3)
2. **Post-Clone Update**: After cloning, the provider calls `config.Update()` to apply Terraform configuration to the new VM
3. **Disk Configuration Build**: During update, `disk.SDK(d)` builds disk configuration from Terraform state
4. **Default Deletion**: Functions like `sdk_Disks_QemuIdeDisksDefault()` mark ALL unused disk slots with `Delete: true`
5. **Cloud-Init Drive Removal**: Since the cloud-init drive isn't explicitly defined in Terraform config, it gets marked for deletion

### Scenario 2: Cloud-Init Parameters Without Explicit Drive
1. **Cloud-Init Parameters**: User specifies cloud-init parameters (`ipconfig0`, `sshkeys`, etc.) in Terraform
2. **Automatic Drive Creation**: Proxmox automatically creates a cloud-init drive (usually on ide2 or ide3) to store this configuration
3. **Post-Clone Update**: During the update after cloning, the provider applies Terraform configuration
4. **Drive Deletion**: Since the auto-created cloud-init drive isn't explicitly defined in the `disks` block, it gets marked for deletion

## The Fix

The fix introduces a comprehensive mechanism to preserve cloud-init drives during VM updates:

### 1. New Preservation Function (`sdk_disks_cloudinit_fix.go`)
```go
// PreserveCloudInitDrive checks if a cloud-init drive exists in the current VM configuration
// and preserves it by not marking it for deletion during updates.
// This handles two scenarios:
// 1. Cloud-init drives inherited from templates during cloning
// 2. Cloud-init drives automatically created by Proxmox when cloud-init parameters are set
func PreserveCloudInitDrive(ctx context.Context, client *pveAPI.Client, vmr *pveAPI.VmRef, storages *pveAPI.QemuStorages, hasCloudInitParams bool) error
```

This function:
- Fetches the current VM configuration from Proxmox
- Checks all disk slots (IDE, SATA, SCSI) for cloud-init drives
- If a cloud-init drive is found (identified by "cloudinit" in the disk string), it sets the storage to `nil` to prevent deletion
- When cloud-init parameters are present, explicitly preserves common cloud-init drive locations (ide2, ide3)

### 2. Integration in Clone Process (`resource_vm_qemu.go`)
```go
log.Print("[DEBUG][QemuVmCreate] update VM after clone")
// Preserve cloud-init drives that exist from the template or are created by Proxmox
if config.Disks != nil {
    // Check if cloud-init parameters are present
    hasCloudInitParams := config.CloudInit != nil || 
        d.Get("ipconfig0").(string) != "" || d.Get("sshkeys").(string) != "" ||
        d.Get("ciuser").(string) != "" || d.Get("cipassword").(string) != "" ||
        // ... other cloud-init parameters ...
    
    err = disk.PreserveCloudInitDrive(ctx, client, vmr, config.Disks, hasCloudInitParams)
    if err != nil {
        log.Printf("[WARN][QemuVmCreate] Failed to preserve cloud-init drive: %v", err)
    }
}
rebootRequired, err = config.Update(ctx, false, vmr, client)
```

The preservation function:
- Detects if cloud-init parameters are present in the Terraform configuration
- Preserves any existing cloud-init drives before the update
- Handles both template-based and auto-created cloud-init drives

## Files Modified

1. **New File**: `proxmox/Internal/resource/guest/qemu/disk/sdk_disks_cloudinit_fix.go`
   - Contains the preservation logic
   - Handles IDE, SATA, and SCSI disk types

2. **New File**: `proxmox/Internal/resource/guest/qemu/disk/sdk_disks_cloudinit_fix_test.go`
   - Unit tests for the preservation logic
   - Tests various scenarios including different cloud-init formats

3. **Modified**: `proxmox/resource_vm_qemu.go`
   - Integrated the preservation function in the clone process
   - Added warning log if preservation fails (non-fatal)

## Benefits

1. **Backward Compatible**: Doesn't change existing behavior for non-cloud-init disks
2. **Non-Invasive**: Only affects VMs with cloud-init drives
3. **Graceful Failure**: If preservation fails, logs a warning but continues
4. **Comprehensive**: Handles cloud-init drives on any disk slot (IDE, SATA, SCSI)

## Testing

The fix includes comprehensive unit tests that verify:
- Cloud-init drives are properly identified and preserved
- Non-cloud-init drives are not affected
- Different cloud-init disk formats are handled correctly
- Edge cases (nil storage, missing disk slots) are handled

## Usage

No changes are required to existing Terraform configurations. The fix automatically preserves cloud-init drives during cloning operations.

### Example Configuration (No Changes Needed)
```hcl
resource "proxmox_vm_qemu" "example" {
  name        = "test-vm"
  target_node = "pve-node"
  clone       = "template-with-cloudinit"
  full_clone  = true
  
  # Cloud-init settings
  ipconfig0   = "ip=10.0.0.10/24,gw=10.0.0.1"
  nameserver  = "8.8.8.8"
  sshkeys     = file("~/.ssh/id_rsa.pub")
  
  # No need to explicitly define the cloud-init drive
  # It will be preserved from the template
}
```

## Alternative Workarounds (Before Fix)

Users experiencing this issue can work around it by:

1. **Explicitly defining the cloud-init drive** in the Terraform configuration:
```hcl
disks {
  ide {
    ide3 {
      cloudinit {
        storage = "local-lvm"
      }
    }
  }
}
```

2. **Using the legacy `cloudinit_cdrom_storage` parameter** (deprecated):
```hcl
cloudinit_cdrom_storage = "local-lvm"
```

With this fix, these workarounds are no longer necessary.
