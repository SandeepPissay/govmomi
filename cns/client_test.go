// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package cns

import (
	"context"
	"github.com/vmware/govmomi/object"
	"os"
	"testing"

	"github.com/vmware/govmomi/find"
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
	//spbmPolicyId4Reconfig := os.Getenv("CNS_SPBM_POLICY_ID_4_RECONFIG")
	//resourcePoolPath := os.Getenv("CNS_RESOURCE_POOL_PATH") // example "/datacenter-name/host/host-ip/Resources" or  /datacenter-name/host/cluster-name/Resources
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

	m := object.NewVirtualDiskManager(c.Client)
	//vdSpec := vim25types.FileBackedVirtualDiskSpec{
	//	VirtualDiskSpec: vim25types.VirtualDiskSpec{
	//		DiskType:    "thin",
	//		AdapterType: "ide",
	//	},
	//	CapacityKb: 10000000,
	//	Profile: []vim25types.BaseVirtualMachineProfileSpec{
	//		&vim25types.VirtualMachineDefinedProfileSpec{
	//			ProfileId: spbmPolicyId4Reconfig,
	//		},
	//	},
	//}
	/*parent := "image/image1.vmdk"
	imgDiskTask, err := m.CreateVirtualDisk(ctx, ds.Path(parent), dc, &vdSpec)
	if err != nil {
		t.Errorf("Error creating a image disk. err: %+v", err)
		t.Fatal(err)
	}
	imgDiskTaskInfo, err := imgDiskTask.WaitForResult(ctx)
	if err != nil {
		t.Errorf("Error waiting for result to get imgDiskTask's info. err: %+v", err)
		t.Fatal(err)
	}
	t.Logf("imgDiskTaskInfo: %+v", imgDiskTaskInfo)*/
	child1 := "new_vm1/new_vm1_disk1.vmdk"
	chld1DiskTask, err := m.CreateChildDisk(ctx, ds.Path("cache1/disk1.vmdk"), dc, ds.Path(child1), dc, true)
	if err != nil {
		t.Errorf("Error creating a child disk. err: %+v", err)
		t.Fatal(err)
	}
	chld1DiskTaskInfo, err := chld1DiskTask.WaitForResult(ctx)
	if err != nil {
		t.Errorf("Error waiting for result to get chld1DiskTask's info. err: %+v", err)
		t.Fatal(err)
	}
	t.Logf("chld1DiskTaskInfo: %+v", chld1DiskTaskInfo)
}
