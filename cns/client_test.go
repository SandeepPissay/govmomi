// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package cns

import (
	"context"
	"os"
	"testing"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/debug"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/vmware/govmomi"
	vim25types "github.com/vmware/govmomi/vim25/types"
)

func TestClient(t *testing.T) {
	// set CNS_DEBUG to true if you need to emit soap traces from these tests
	// soap traces will be emitted in the govmomi/cns/.soap directory
	// example export CNS_DEBUG='true'
	enableDebug := os.Getenv("CNS_DEBUG")
	soapTraceDirectory := ".soap"

	url := os.Getenv("CNS_VC_URL") // example: export CNS_VC_URL='https://username:password@vc-ip/sdk'
	datacenter := os.Getenv("CNS_DATACENTER")
	datastore := os.Getenv("CNS_DATASTORE")

	if url == "" || datacenter == "" || datastore == "" {
		t.Skip("CNS_VC_URL or CNS_DATACENTER or CNS_DATASTORE is not set")
	}
	resourcePoolPath := os.Getenv("CNS_RESOURCE_POOL_PATH") // example "/datacenter-name/host/host-ip/Resources" or  /datacenter-name/host/cluster-name/Resources
	u, err := soap.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}

	if enableDebug == "true" {
		if _, err := os.Stat(soapTraceDirectory); os.IsNotExist(err) {
			os.Mkdir(soapTraceDirectory, 0755)
		}
		p := debug.FileProvider{
			Path: soapTraceDirectory,
		}
		debug.SetProvider(&p)
	}

	ctx := context.Background()
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		t.Fatal(err)
	}
	cnsClient, err := NewClient(ctx, c.Client)
	if err != nil {
		t.Fatal(err)
	}
	finder := find.NewFinder(cnsClient.vim25Client, false)
	dc, err := finder.Datacenter(ctx, datacenter)
	if err != nil {
		t.Fatal(err)
	}
	finder.SetDatacenter(dc)
	ds, err := finder.Datastore(ctx, datastore)
	if err != nil {
		t.Fatal(err)
	}

	props := []string{"info", "summary"}
	pc := property.DefaultCollector(c.Client)
	var dsSummaries []mo.Datastore
	err = pc.Retrieve(ctx, []vim25types.ManagedObjectReference{ds.Reference()}, props, &dsSummaries)
	if err != nil {
		t.Fatal(err)
	}
	dsUrl := dsSummaries[0].Summary.Url
	t.Logf("dsUrl: %+v", dsUrl)

	unit := int32(0)
	thin := true
	var dsRef vim25types.ManagedObjectReference
	dsRef = ds.Reference()

	// Create a VM to test Attach Volume API.
	virtualMachineConfigSpec := vim25types.VirtualMachineConfigSpec{
		Name: "test-node-vm",
		Files: &vim25types.VirtualMachineFileInfo{
			VmPathName: "[" + datastore + "]",
		},
		NumCPUs:  1,
		MemoryMB: 4,
		DeviceChange: []vim25types.BaseVirtualDeviceConfigSpec{
			&vim25types.VirtualDeviceConfigSpec{
				Operation: vim25types.VirtualDeviceConfigSpecOperationAdd,
				Device: &vim25types.ParaVirtualSCSIController{
					VirtualSCSIController: vim25types.VirtualSCSIController{
						SharedBus: vim25types.VirtualSCSISharingNoSharing,
						VirtualController: vim25types.VirtualController{
							BusNumber: 0,
							VirtualDevice: vim25types.VirtualDevice{
								Key: 1000,
							},
							//Device: []int32{2000},
						},
					},
				},
			},
			&vim25types.VirtualDeviceConfigSpec{
				Operation:     vim25types.VirtualDeviceConfigSpecOperationAdd,
				FileOperation: vim25types.VirtualDeviceConfigSpecFileOperationCreate,
				Device: &vim25types.VirtualDisk{
					VirtualDevice: vim25types.VirtualDevice{
						Key: 2000,
						Backing: &vim25types.VirtualDiskFlatVer2BackingInfo{
							VirtualDeviceFileBackingInfo: vim25types.VirtualDeviceFileBackingInfo{
								FileName: "[" + datastore + "] test-node-vm/test-node-vm.vmdk",
							},
							DiskMode:        string(vim25types.VirtualDiskModePersistent),
							ThinProvisioned: &thin,
						},

						ControllerKey: 1000,
						UnitNumber:    &unit,
					},
					CapacityInBytes: 10 * 1024 * 1024,
					CapacityInKB:    10 * 1024,
				},
			},
		},
	}
	defaultFolder, err := finder.DefaultFolder(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var resourcePool *object.ResourcePool
	if resourcePoolPath == "" {
		resourcePool, err = finder.DefaultResourcePool(ctx)
	} else {
		resourcePool, err = finder.ResourcePool(ctx, resourcePoolPath)
	}
	if err != nil {
		t.Errorf("Error occurred while getting DefaultResourcePool. err: %+v", err)
		t.Fatal(err)
	}
	task, err := defaultFolder.CreateVM(ctx, virtualMachineConfigSpec, resourcePool, nil)
	if err != nil {
		t.Errorf("Failed to create VM. Error: %+v \n", err)
		t.Fatal(err)
	}

	vmTaskInfo, err := task.WaitForResult(ctx, nil)
	if err != nil {
		t.Errorf("Error occurred while waiting for create VM task result. err: %+v", err)
		t.Fatal(err)
	}

	vmRef := vmTaskInfo.Result.(object.Reference)
	t.Logf("Node VM created sucessfully. vmRef: %+v", vmRef.Reference())

	vm, err := finder.VirtualMachine(ctx, "test-node-vm")
	if err != nil {
		t.Errorf("Error finding the vm. err: %+v", err)
		t.Fatal(err)
	}
	snapTask, err := vm.CreateSnapshot(ctx, "snap-1", "snap-desc1", false, false)
	if err != nil {
		t.Errorf("Error snapshotting the vm. err: %+v", err)
		t.Fatal(err)
	}
	vmSnapTaskInfo, err := snapTask.WaitForResult(ctx)
	if err != nil {
		t.Errorf("Error waiting for vm snapshot task. err: %+v", err)
		t.Fatal(err)
	}
	vmSnapRef := vmSnapTaskInfo.Result.(object.Reference)
	t.Logf("vmSnapRef : %+v", vmSnapRef)

	var vmSnapRef2 vim25types.ManagedObjectReference
	vmSnapRef2 = vmSnapRef.Reference()
	vmCloneSpec := &vim25types.VirtualMachineCloneSpec{
		Location: vim25types.VirtualMachineRelocateSpec{
			DiskMoveType: string(vim25types.VirtualMachineRelocateDiskMoveOptionsCreateNewChildDiskBacking),
			Datastore:    &dsRef,
		},
		Snapshot: &vmSnapRef2,
		PowerOn:  false,
	}
	lcTask, err := vm.Clone(ctx, defaultFolder, "clone-vm", *vmCloneSpec)
	if err != nil {
		t.Logf("Link clone failed. err: %+v", err)
		t.Fatal(err)
	}
	lcTaskInfo, err := lcTask.WaitForResult(ctx)
	if err != nil {
		t.Errorf("Error waiting for link clone vm task. err: %+v", err)
		t.Fatal(err)
	}
	lcVmRef := lcTaskInfo.Result.(object.Reference)
	t.Logf("lcVmRef : %+v", lcVmRef)

	//nodeVM := object.NewVirtualMachine(cnsClient.vim25Client, vmRef.Reference())
	//defer nodeVM.Destroy(ctx)

}
