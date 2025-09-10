# Cloud-Init Drive Deletion Bug Fix for v3.0.1

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

## The Fix Applied to v3.0.1

This fix has been backported from the master branch to the v3.0.1 branch (based on tag v3.0.1-rc10).

### Changes Made

1. **New File**: `proxmox/Internal/resource/guest/qemu/disk/sdk_disks_cloudinit_fix.go`
   - Contains logic to preserve cloud-init drives
   - Handles both inherited and auto-created drives
   - Checks IDE, SATA, and SCSI disk slots

2. **Modified**: `proxmox/resource_vm_qemu.go`
   - Added cloud-init preservation logic before `config.Update()` call
   - Detects presence of cloud-init parameters
   - Calls preservation function with appropriate flags

### How It Works

The fix:
1. Fetches the current VM configuration from Proxmox
2. Checks all disk slots for cloud-init drives (identified by "cloudinit" in the disk string)
3. If cloud-init parameters are present in Terraform config, also preserves common cloud-init slots (ide2, ide3)
4. Sets preserved drives to `nil` instead of marking them for deletion

## Usage

No changes are required to existing Terraform configurations. The fix automatically preserves cloud-init drives during cloning operations.

### Example Configuration
```hcl
resource "proxmox_vm_qemu" "example" {
  name        = "test-vm"
  target_node = "pve-node"
  clone       = "template-with-or-without-cloudinit"
  full_clone  = true
  
  # Cloud-init settings
  ipconfig0   = "ip=10.0.0.10/24,gw=10.0.0.1"
  nameserver  = "8.8.8.8"
  sshkeys     = file("~/.ssh/id_rsa.pub")
  
  # No need to explicitly define the cloud-init drive
  # It will be preserved automatically
}
```

## Testing

To test this fix:
1. Create a VM using a template (with or without cloud-init drive)
2. Specify cloud-init parameters in your Terraform configuration
3. Apply the configuration
4. Verify the cloud-init drive is preserved and functional

## Branch Information

- **Branch**: v3.0.1
- **Base Tag**: v3.0.1-rc10
- **Fix Applied**: Cloud-init drive preservation during VM clone operations
