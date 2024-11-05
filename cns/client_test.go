/*
Copyright (c) 2019 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cns

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/debug"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/vmware/govmomi"
	cnstypes "github.com/vmware/govmomi/cns/types"
	vim25types "github.com/vmware/govmomi/vim25/types"
	vsanfstypes "github.com/vmware/govmomi/vsan/vsanfs/types"
)

const VSphere70u3VersionInt = 703

func TestClient(t *testing.T) {
	// set CNS_DEBUG to true if you need to emit soap traces from these tests
	// soap traces will be emitted in the govmomi/cns/.soap directory
	// example export CNS_DEBUG='true'
	enableDebug := os.Getenv("CNS_DEBUG")
	soapTraceDirectory := ".soap"

	url := os.Getenv("CNS_VC_URL") // example: export CNS_VC_URL='https://username:password@vc-ip/sdk'
	datacenter := os.Getenv("CNS_DATACENTER")
	datastore := os.Getenv("CNS_DATASTORE")

	// set CNS_RUN_FILESHARE_TESTS environment to true, if your setup has vsanfileshare enabled.
	// when CNS_RUN_FILESHARE_TESTS is not set to true, vsan file share related tests are skipped.
	// example: export CNS_RUN_FILESHARE_TESTS='true'
	//run_fileshare_tests := os.Getenv("CNS_RUN_FILESHARE_TESTS")

	// if backingDiskURLPath is not set, test for Creating Volume with setting BackingDiskUrlPath in the BackingObjectDetails of
	// CnsVolumeCreateSpec will be skipped.
	// example: export BACKING_DISK_URL_PATH='https://vc-ip/folder/vmdkfilePath.vmdk?dcPath=DataCenterPath&dsName=DataStoreName'
	//backingDiskURLPath := os.Getenv("BACKING_DISK_URL_PATH")

	// if datastoreForMigration is not set, test for CNS Relocate API of a volume to another datastore is skipped.
	// input format is same as CNS_DATASTORE. Format eg. "vSANDirect_10.92.217.162_mpx.vmhba0:C0:T2:L0"/ "vsandatastore"
	// make sure that migration datastore is accessible from host on which CNS_DATASTORE is mounted.
	//datastoreForMigration := os.Getenv("CNS_MIGRATION_DATASTORE")

	// if spbmPolicyId4Reconfig is not set, test for CnsReconfigVolumePolicy API will be skipped
	// example: export CNS_SPBM_POLICY_ID_4_RECONFIG=6f64d90e-2ad5-4c4d-8cbc-a3330ebc496c
	//spbmPolicyId4Reconfig := os.Getenv("CNS_SPBM_POLICY_ID_4_RECONFIG")

	if url == "" || datacenter == "" || datastore == "" {
		t.Skip("CNS_VC_URL or CNS_DATACENTER or CNS_DATASTORE is not set")
	}
	//resporcePoolPath := os.Getenv("CNS_RESOURCE_POOL_PATH") // example "/datacenter-name/host/host-ip/Resources" or  /datacenter-name/host/cluster-name/Resources
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
	// UseServiceVersion sets soap.Client.Version to the current version of the service endpoint via /sdk/vsanServiceVersions.xml
	c.UseServiceVersion("vsan")
	cnsClient, err := NewClient(ctx, c.Client)
	if err != nil {
		t.Fatal(err)
	}
	finder := find.NewFinder(cnsClient.Vim25Client, false)
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
	//dsUrl := dsSummaries[0].Summary.Url

	var dsList []vim25types.ManagedObjectReference
	dsList = append(dsList, ds.Reference())

	var containerClusterArray []cnstypes.CnsContainerCluster
	containerCluster := cnstypes.CnsContainerCluster{
		ClusterType:         string(cnstypes.CnsClusterTypeKubernetes),
		ClusterId:           "demo-cluster-id",
		VSphereUser:         "Administrator@vsphere.local",
		ClusterFlavor:       string(cnstypes.CnsClusterFlavorVanilla),
		ClusterDistribution: "DemoCluster",
	}
	containerClusterArray = append(containerClusterArray, containerCluster)

	var cnsFileVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	vSANFileCreateSpec := &cnstypes.CnsVSANFileCreateSpec{
		SoftQuotaInMb: 5120,
		Permission: []vsanfstypes.VsanFileShareNetPermission{
			{
				Ips:         "*",
				Permissions: vsanfstypes.VsanFileShareAccessTypeREAD_WRITE,
				AllowRoot:   true,
			},
		},
	}

	cnsFileVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
		Name:       "cns-file-volume",
		VolumeType: string(cnstypes.CnsVolumeTypeFile),
		Datastores: dsList,
		Metadata: cnstypes.CnsVolumeMetadata{
			ContainerCluster:      containerCluster,
			ContainerClusterArray: containerClusterArray,
		},
		BackingObjectDetails: &cnstypes.CnsVsanFileShareBackingDetails{
			CnsFileBackingDetails: cnstypes.CnsFileBackingDetails{
				CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
					CapacityInMb: 5120,
				},
			},
		},
		CreateSpec: vSANFileCreateSpec,
	}
	cnsFileVolumeCreateSpecList = append(cnsFileVolumeCreateSpecList, cnsFileVolumeCreateSpec)
	t.Logf("Creating CNS file volume using the spec: %+v", cnsFileVolumeCreateSpec)
	createTask, err := cnsClient.CreateVolume(ctx, cnsFileVolumeCreateSpecList)
	if err != nil {
		t.Errorf("Failed to create vsan fileshare volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	createTaskInfo, err := GetTaskInfo(ctx, createTask)
	if err != nil {
		t.Errorf("Failed to create Fileshare volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	createTaskResult, err := GetTaskResult(ctx, createTaskInfo)
	if err != nil {
		t.Errorf("Failed to create Fileshare volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	if createTaskResult == nil {
		t.Fatalf("Empty create task results")
		t.FailNow()
	}
	createVolumeOperationRes := createTaskResult.GetCnsVolumeOperationResult()
	if createVolumeOperationRes.Fault != nil {
		t.Fatalf("Failed to create Fileshare volume: fault=%+v", createVolumeOperationRes.Fault)
	}
	filevolumeId := createVolumeOperationRes.VolumeId.Id
	t.Logf("Fileshare volume created sucessfully. filevolumeId: %s", filevolumeId)

	// Test CreateVolume API
	//var cnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	//cnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
	//	Name:       "pvc-901e87eb-c2bd-11e9-806f-005056a0c9a0",
	//	VolumeType: string(cnstypes.CnsVolumeTypeBlock),
	//	Datastores: dsList,
	//	Metadata: cnstypes.CnsVolumeMetadata{
	//		ContainerCluster: containerCluster,
	//	},
	//	BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
	//		CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
	//			CapacityInMb: 5120,
	//		},
	//	},
	//}
	//cnsVolumeCreateSpecList = append(cnsVolumeCreateSpecList, cnsVolumeCreateSpec)
	//t.Logf("Creating volume using the spec: %+v", pretty.Sprint(cnsVolumeCreateSpec))
	//createTask, err := cnsClient.CreateVolume(ctx, cnsVolumeCreateSpecList)
	//if err != nil {
	//	t.Errorf("Failed to create volume. Error: %+v \n", err)
	//	t.Fatal(err)
	//}
	//createTaskInfo, err := GetTaskInfo(ctx, createTask)
	//if err != nil {
	//	t.Errorf("Failed to create volume. Error: %+v \n", err)
	//	t.Fatal(err)
	//}
	//createTaskResult, err := GetTaskResult(ctx, createTaskInfo)
	//if err != nil {
	//	t.Errorf("Failed to create volume. Error: %+v \n", err)
	//	t.Fatal(err)
	//}
	//if createTaskResult == nil {
	//	t.Fatalf("Empty create task results")
	//	t.FailNow()
	//}
	//createVolumeOperationRes := createTaskResult.GetCnsVolumeOperationResult()
	//if createVolumeOperationRes.Fault != nil {
	//	t.Fatalf("Failed to create volume: fault=%+v", createVolumeOperationRes.Fault)
	//}
	//volumeId := createVolumeOperationRes.VolumeId.Id
	//volumeCreateResult := (createTaskResult).(*cnstypes.CnsVolumeCreateResult)
	//t.Logf("volumeCreateResult %+v", volumeCreateResult)
	//t.Logf("Volume created sucessfully. volumeId: %s", volumeId)
	//
	//if cnsClient.Version != ReleaseVSAN67u3 {
	//	// Test creating static volume using existing CNS volume should fail
	//	var staticCnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	//	staticCnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
	//		Name:       "pvc-901e87eb-c2bd-11e9-806f-005056a0c9a0",
	//		VolumeType: string(cnstypes.CnsVolumeTypeBlock),
	//		Metadata: cnstypes.CnsVolumeMetadata{
	//			ContainerCluster: containerCluster,
	//		},
	//		BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
	//			CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
	//				CapacityInMb: 5120,
	//			},
	//			BackingDiskId: volumeId,
	//		},
	//	}
	//
	//	staticCnsVolumeCreateSpecList = append(staticCnsVolumeCreateSpecList, staticCnsVolumeCreateSpec)
	//	t.Logf("Creating volume using the spec: %+v", pretty.Sprint(staticCnsVolumeCreateSpec))
	//	recreateTask, err := cnsClient.CreateVolume(ctx, staticCnsVolumeCreateSpecList)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	reCreateTaskInfo, err := GetTaskInfo(ctx, recreateTask)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	reCreateTaskResult, err := GetTaskResult(ctx, reCreateTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	if reCreateTaskResult == nil {
	//		t.Fatalf("Empty create task results")
	//		t.FailNow()
	//	}
	//	reCreateVolumeOperationRes := reCreateTaskResult.GetCnsVolumeOperationResult()
	//	t.Logf("reCreateVolumeOperationRes.: %+v", pretty.Sprint(reCreateVolumeOperationRes))
	//	if reCreateVolumeOperationRes.Fault != nil {
	//		t.Logf("reCreateVolumeOperationRes.Fault: %+v", pretty.Sprint(reCreateVolumeOperationRes.Fault))
	//		_, ok := reCreateVolumeOperationRes.Fault.Fault.(cnstypes.CnsAlreadyRegisteredFault)
	//		if !ok {
	//			t.Fatalf("Fault is not a CnsAlreadyRegisteredFault")
	//		}
	//	} else {
	//		t.Fatalf("re-create same volume should fail with CnsAlreadyRegisteredFault")
	//	}
	//}

	//if run_fileshare_tests == "true" && cnsClient.Version != ReleaseVSAN67u3 {
	//	// Test creating vSAN file-share Volume
	//	var cnsFileVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	//	vSANFileCreateSpec := &cnstypes.CnsVSANFileCreateSpec{
	//		SoftQuotaInMb: 5120,
	//		Permission: []vsanfstypes.VsanFileShareNetPermission{
	//			{
	//				Ips:         "*",
	//				Permissions: vsanfstypes.VsanFileShareAccessTypeREAD_WRITE,
	//				AllowRoot:   true,
	//			},
	//		},
	//	}
	//
	//	cnsFileVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
	//		Name:       "pvc-file-share-volume",
	//		VolumeType: string(cnstypes.CnsVolumeTypeFile),
	//		Datastores: dsList,
	//		Metadata: cnstypes.CnsVolumeMetadata{
	//			ContainerCluster:      containerCluster,
	//			ContainerClusterArray: containerClusterArray,
	//		},
	//		BackingObjectDetails: &cnstypes.CnsVsanFileShareBackingDetails{
	//			CnsFileBackingDetails: cnstypes.CnsFileBackingDetails{
	//				CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
	//					CapacityInMb: 5120,
	//				},
	//			},
	//		},
	//		CreateSpec: vSANFileCreateSpec,
	//	}
	//	cnsFileVolumeCreateSpecList = append(cnsFileVolumeCreateSpecList, cnsFileVolumeCreateSpec)
	//	t.Logf("Creating CNS file volume using the spec: %+v", cnsFileVolumeCreateSpec)
	//	createTask, err = cnsClient.CreateVolume(ctx, cnsFileVolumeCreateSpecList)
	//	if err != nil {
	//		t.Errorf("Failed to create vsan fileshare volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	createTaskInfo, err = GetTaskInfo(ctx, createTask)
	//	if err != nil {
	//		t.Errorf("Failed to create Fileshare volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	createTaskResult, err = GetTaskResult(ctx, createTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to create Fileshare volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	if createTaskResult == nil {
	//		t.Fatalf("Empty create task results")
	//		t.FailNow()
	//	}
	//	createVolumeOperationRes = createTaskResult.GetCnsVolumeOperationResult()
	//	if createVolumeOperationRes.Fault != nil {
	//		t.Fatalf("Failed to create Fileshare volume: fault=%+v", createVolumeOperationRes.Fault)
	//	}
	//	filevolumeId := createVolumeOperationRes.VolumeId.Id
	//	t.Logf("Fileshare volume created sucessfully. filevolumeId: %s", filevolumeId)
	//
	//	// Test QueryVolume API
	//	volumeIDList = []cnstypes.CnsVolumeId{{Id: filevolumeId}}
	//	queryFilter.VolumeIds = volumeIDList
	//	t.Logf("Calling QueryVolume using queryFilter: %+v", queryFilter)
	//	queryResult, err = cnsClient.QueryVolume(ctx, queryFilter)
	//	if err != nil {
	//		t.Errorf("Failed to query volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	t.Logf("Successfully Queried Volumes. queryResult: %+v", queryResult)
	//	fileBackingInfo := queryResult.Volumes[0].BackingObjectDetails.(*cnstypes.CnsVsanFileShareBackingDetails)
	//	t.Logf("File Share Name: %s with accessPoints: %+v", fileBackingInfo.Name, fileBackingInfo.AccessPoints)
	//
	//	// Test add read-only permissions using Configure ACLs
	//	netPerms := make([]vsanfstypes.VsanFileShareNetPermission, 0)
	//	netPerms = append(netPerms, vsanfstypes.VsanFileShareNetPermission{
	//		Ips:         "192.168.124.2",
	//		Permissions: "READ_ONLY",
	//	})
	//
	//	vSanNFSACLEntry := make([]cnstypes.CnsNFSAccessControlSpec, 0)
	//	vSanNFSACLEntry = append(vSanNFSACLEntry, cnstypes.CnsNFSAccessControlSpec{
	//		Permission: netPerms,
	//	})
	//
	//	volumeID := cnstypes.CnsVolumeId{
	//		Id: filevolumeId,
	//	}
	//	aclSpec := cnstypes.CnsVolumeACLConfigureSpec{
	//		VolumeId:              volumeID,
	//		AccessControlSpecList: vSanNFSACLEntry,
	//	}
	//	t.Logf("Invoking ConfigureVolumeACLs using the spec: %+v", pretty.Sprint(aclSpec))
	//	aclTask, err := cnsClient.ConfigureVolumeACLs(ctx, aclSpec)
	//	if err != nil {
	//		t.Errorf("Failed to configure VolumeACLs. Error: %+v", err)
	//		t.Fatal(err)
	//	}
	//	aclTaskInfo, err := GetTaskInfo(ctx, aclTask)
	//	if err != nil {
	//		t.Errorf("Failed to configure VolumeACLs. Error: %+v", err)
	//		t.Fatal(err)
	//	}
	//	aclTaskResult, err := GetTaskResult(ctx, aclTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to configure VolumeACLs. Error: %+v", err)
	//		t.Fatal(err)
	//	}
	//	if aclTaskResult == nil {
	//		t.Fatalf("Empty configure VolumeACLs task results")
	//		t.FailNow()
	//	}
	//
	//	// Test to revoke all permissions using Configure ACLs
	//	netPerms = make([]vsanfstypes.VsanFileShareNetPermission, 0)
	//	netPerms = append(netPerms, vsanfstypes.VsanFileShareNetPermission{
	//		Ips:         "192.168.124.2",
	//		Permissions: "READ_ONLY",
	//	})
	//
	//	vSanNFSACLEntry = make([]cnstypes.CnsNFSAccessControlSpec, 0)
	//	vSanNFSACLEntry = append(vSanNFSACLEntry, cnstypes.CnsNFSAccessControlSpec{
	//		Permission: netPerms,
	//		Delete:     true,
	//	})
	//
	//	aclSpec = cnstypes.CnsVolumeACLConfigureSpec{
	//		VolumeId:              volumeID,
	//		AccessControlSpecList: vSanNFSACLEntry,
	//	}
	//	t.Logf("Invoking ConfigureVolumeACLs using the spec: %+v", pretty.Sprint(aclSpec))
	//	aclTask, err = cnsClient.ConfigureVolumeACLs(ctx, aclSpec)
	//	if err != nil {
	//		t.Errorf("Failed to configure VolumeACLs. Error: %+v", err)
	//		t.Fatal(err)
	//	}
	//	aclTaskInfo, err = GetTaskInfo(ctx, aclTask)
	//	if err != nil {
	//		t.Errorf("Failed to configure VolumeACLs. Error: %+v", err)
	//		t.Fatal(err)
	//	}
	//	aclTaskResult, err = GetTaskResult(ctx, aclTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to configure VolumeACLs. Error: %+v", err)
	//		t.Fatal(err)
	//	}
	//	if aclTaskResult == nil {
	//		t.Fatalf("Empty configure VolumeACLs task results")
	//		t.FailNow()
	//	}
	//
	//	// Test Deleting vSAN file-share Volume
	//	var fileVolumeIDList []cnstypes.CnsVolumeId
	//	fileVolumeIDList = append(fileVolumeIDList, cnstypes.CnsVolumeId{Id: filevolumeId})
	//	t.Logf("Deleting fileshare volume: %+v", fileVolumeIDList)
	//	deleteTask, err = cnsClient.DeleteVolume(ctx, fileVolumeIDList, true)
	//	if err != nil {
	//		t.Errorf("Failed to delete fileshare volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	deleteTaskInfo, err = GetTaskInfo(ctx, deleteTask)
	//	if err != nil {
	//		t.Errorf("Failed to delete fileshare volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	deleteTaskResult, err = GetTaskResult(ctx, deleteTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to delete fileshare volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	if deleteTaskResult == nil {
	//		t.Fatalf("Empty delete task results")
	//		t.FailNow()
	//	}
	//	deleteVolumeOperationRes = deleteTaskResult.GetCnsVolumeOperationResult()
	//	if deleteVolumeOperationRes.Fault != nil {
	//		t.Fatalf("Failed to delete fileshare volume: fault=%+v", deleteVolumeOperationRes.Fault)
	//	}
	//	t.Logf("fileshare volume:%q deleted sucessfully", filevolumeId)
	//}
	//if backingDiskURLPath != "" && cnsClient.Version != ReleaseVSAN67u3 && cnsClient.Version != ReleaseVSAN70 {
	//	// Test CreateVolume API with existing VMDK
	//	var cnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	//	cnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
	//		Name:       "pvc-901e87eb-c2bd-11e9-806f-005056a0c9a0",
	//		VolumeType: string(cnstypes.CnsVolumeTypeBlock),
	//		Metadata: cnstypes.CnsVolumeMetadata{
	//			ContainerCluster: containerCluster,
	//		},
	//		BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
	//			BackingDiskUrlPath: backingDiskURLPath,
	//		},
	//	}
	//	cnsVolumeCreateSpecList = append(cnsVolumeCreateSpecList, cnsVolumeCreateSpec)
	//	t.Logf("Creating volume using the spec: %+v", pretty.Sprint(cnsVolumeCreateSpec))
	//	createTask, err := cnsClient.CreateVolume(ctx, cnsVolumeCreateSpecList)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	createTaskInfo, err := GetTaskInfo(ctx, createTask)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	createTaskResult, err := GetTaskResult(ctx, createTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	if createTaskResult == nil {
	//		t.Fatalf("Empty create task results")
	//		t.FailNow()
	//	}
	//	createVolumeOperationRes := createTaskResult.GetCnsVolumeOperationResult()
	//	var volumeID string
	//	if createVolumeOperationRes.Fault != nil {
	//		t.Logf("Failed to create volume: fault=%+v", createVolumeOperationRes.Fault)
	//		fault, ok := createVolumeOperationRes.Fault.Fault.(cnstypes.CnsAlreadyRegisteredFault)
	//		if !ok {
	//			t.Fatalf("Fault is not CnsAlreadyRegisteredFault")
	//		} else {
	//			t.Logf("Fault is CnsAlreadyRegisteredFault. backingDiskURLPath: %s is already registered", backingDiskURLPath)
	//			volumeID = fault.VolumeId.Id
	//		}
	//	} else {
	//		volumeID = createVolumeOperationRes.VolumeId.Id
	//		t.Logf("Volume created sucessfully with backingDiskURLPath: %s. volumeId: %s", backingDiskURLPath, volumeID)
	//
	//		// Test re creating volume using BACKING_DISK_URL_PATH
	//		var reCreateCnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	//		reCreateCnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
	//			Name:       "pvc-901e87eb-c2bd-11e9-806f-005056a0c9a0",
	//			VolumeType: string(cnstypes.CnsVolumeTypeBlock),
	//			Metadata: cnstypes.CnsVolumeMetadata{
	//				ContainerCluster: containerCluster,
	//			},
	//			BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
	//				BackingDiskUrlPath: backingDiskURLPath,
	//			},
	//		}
	//
	//		reCreateCnsVolumeCreateSpecList = append(reCreateCnsVolumeCreateSpecList, reCreateCnsVolumeCreateSpec)
	//		t.Logf("Creating volume using the spec: %+v", pretty.Sprint(reCreateCnsVolumeCreateSpec))
	//		recreateTask, err := cnsClient.CreateVolume(ctx, reCreateCnsVolumeCreateSpecList)
	//		if err != nil {
	//			t.Errorf("Failed to create volume. Error: %+v \n", err)
	//			t.Fatal(err)
	//		}
	//		reCreateTaskInfo, err := GetTaskInfo(ctx, recreateTask)
	//		if err != nil {
	//			t.Errorf("Failed to create volume. Error: %+v \n", err)
	//			t.Fatal(err)
	//		}
	//		reCreateTaskResult, err := GetTaskResult(ctx, reCreateTaskInfo)
	//		if err != nil {
	//			t.Errorf("Failed to create volume. Error: %+v \n", err)
	//			t.Fatal(err)
	//		}
	//		if reCreateTaskResult == nil {
	//			t.Fatalf("Empty create task results")
	//			t.FailNow()
	//		}
	//		reCreateVolumeOperationRes := reCreateTaskResult.GetCnsVolumeOperationResult()
	//		t.Logf("reCreateVolumeOperationRes.: %+v", pretty.Sprint(reCreateVolumeOperationRes))
	//		if reCreateVolumeOperationRes.Fault != nil {
	//			t.Logf("Failed to create volume: fault=%+v", reCreateVolumeOperationRes.Fault)
	//			_, ok := reCreateVolumeOperationRes.Fault.Fault.(cnstypes.CnsAlreadyRegisteredFault)
	//			if !ok {
	//				t.Fatalf("Fault is not CnsAlreadyRegisteredFault")
	//			} else {
	//				t.Logf("Fault is CnsAlreadyRegisteredFault. backingDiskURLPath: %q is already registered", backingDiskURLPath)
	//			}
	//		}
	//	}
	//
	//	// Test QueryVolume API
	//	var queryFilter cnstypes.CnsQueryFilter
	//	var volumeIDList []cnstypes.CnsVolumeId
	//	volumeIDList = append(volumeIDList, cnstypes.CnsVolumeId{Id: volumeID})
	//	queryFilter.VolumeIds = volumeIDList
	//	t.Logf("Calling QueryVolume using queryFilter: %+v", pretty.Sprint(queryFilter))
	//	queryResult, err := cnsClient.QueryVolume(ctx, queryFilter)
	//	if err != nil {
	//		t.Errorf("Failed to query volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	t.Logf("Successfully Queried Volumes. queryResult: %+v", pretty.Sprint(queryResult))
	//
	//	t.Logf("Deleting CNS volume created above using BACKING_DISK_URL_PATH: %s with volume: %+v", backingDiskURLPath, volumeIDList)
	//	deleteTask, err = cnsClient.DeleteVolume(ctx, volumeIDList, true)
	//	if err != nil {
	//		t.Errorf("Failed to delete volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	deleteTaskInfo, err = GetTaskInfo(ctx, deleteTask)
	//	if err != nil {
	//		t.Errorf("Failed to delete volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	deleteTaskResult, err = GetTaskResult(ctx, deleteTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to delete volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	if deleteTaskResult == nil {
	//		t.Fatalf("Empty delete task results")
	//		t.FailNow()
	//	}
	//	deleteVolumeOperationRes = deleteTaskResult.GetCnsVolumeOperationResult()
	//	if deleteVolumeOperationRes.Fault != nil {
	//		t.Fatalf("Failed to delete volume: fault=%+v", deleteVolumeOperationRes.Fault)
	//	}
	//	t.Logf("volume:%q deleted sucessfully", volumeID)
	//}
	//
	//// Test CnsReconfigVolumePolicy API
	//if spbmPolicyId4Reconfig != "" {
	//	var cnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	//	cnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
	//		Name:       "pvc-901e87eb-c2bd-11e9-806f-005056a0c9a0-1",
	//		VolumeType: string(cnstypes.CnsVolumeTypeBlock),
	//		Datastores: dsList,
	//		Metadata: cnstypes.CnsVolumeMetadata{
	//			ContainerCluster: containerCluster,
	//		},
	//		BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
	//			CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
	//				CapacityInMb: 5120,
	//			},
	//		},
	//	}
	//	cnsVolumeCreateSpecList = append(cnsVolumeCreateSpecList, cnsVolumeCreateSpec)
	//	t.Logf("Creating volume using the spec: %+v", pretty.Sprint(cnsVolumeCreateSpecList))
	//	createTask, err = cnsClient.CreateVolume(ctx, cnsVolumeCreateSpecList)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	createTaskInfo, err = GetTaskInfo(ctx, createTask)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	createTaskResult, err = GetTaskResult(ctx, createTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to create volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	if createTaskResult == nil {
	//		t.Fatalf("Empty create task results")
	//		t.FailNow()
	//	}
	//	createVolumeOperationRes = createTaskResult.GetCnsVolumeOperationResult()
	//	if createVolumeOperationRes.Fault != nil {
	//		t.Fatalf("Failed to create volume: fault=%+v", createVolumeOperationRes.Fault)
	//	}
	//	volumeId = createVolumeOperationRes.VolumeId.Id
	//	volumeCreateResult = (createTaskResult).(*cnstypes.CnsVolumeCreateResult)
	//	t.Logf("volumeCreateResult %+v", volumeCreateResult)
	//	t.Logf("Volume created sucessfully. volumeId: %s", volumeId)
	//
	//	t.Logf("Calling reconfigpolicy on volume %v with policy %+v \n", volumeId, spbmPolicyId4Reconfig)
	//	reconfigSpecs := []cnstypes.CnsVolumePolicyReconfigSpec{
	//		{
	//			VolumeId: createVolumeOperationRes.VolumeId,
	//			Profile: []vim25types.BaseVirtualMachineProfileSpec{
	//				&vim25types.VirtualMachineDefinedProfileSpec{
	//					ProfileId: spbmPolicyId4Reconfig,
	//				},
	//			},
	//		},
	//	}
	//	reconfigTask, err := cnsClient.ReconfigVolumePolicy(ctx, reconfigSpecs)
	//	if err != nil {
	//		t.Errorf("Failed to reconfig policy %v on volume %v. Error: %+v \n", spbmPolicyId4Reconfig, volumeId, err)
	//		t.Fatal(err)
	//	}
	//	reconfigTaskInfo, err := GetTaskInfo(ctx, reconfigTask)
	//	if err != nil {
	//		t.Errorf("Failed to reconfig volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	reconfigTaskResult, err := GetTaskResult(ctx, reconfigTaskInfo)
	//	if err != nil {
	//		t.Errorf("Failed to reconfig volume. Error: %+v \n", err)
	//		t.Fatal(err)
	//	}
	//	if reconfigTaskResult == nil {
	//		t.Fatalf("Empty reconfig task results")
	//		t.FailNow()
	//	}
	//	reconfigVolumeOperationRes := reconfigTaskResult.GetCnsVolumeOperationResult()
	//	if reconfigVolumeOperationRes.Fault != nil {
	//		t.Fatalf("Failed to reconfig volume %v with policy %v: fault=%+v",
	//			volumeId, spbmPolicyId4Reconfig, reconfigVolumeOperationRes.Fault)
	//	}
	//	t.Logf("reconfigpolicy on volume %v with policy %+v successful\n", volumeId, spbmPolicyId4Reconfig)
	//}
	//
	//// Test CnsSyncDatastore API
	//t.Logf("Calling syncDatastore on %v ...\n", dsUrl)
	//syncDatastoreTask, err := cnsClient.SyncDatastore(ctx, dsUrl, false)
	//if err != nil {
	//	t.Errorf("Failed to sync datastore %v. Error: %+v \n", dsUrl, err)
	//	t.Fatal(err)
	//}
	//syncDatastoreTaskInfo, err := GetTaskInfo(ctx, syncDatastoreTask)
	//if err != nil {
	//	t.Errorf("Failed to get sync datastore taskInfo. Error: %+v \n", err)
	//	t.Fatal(err)
	//}
	//if syncDatastoreTaskInfo.State != vim25types.TaskInfoStateSuccess {
	//	t.Errorf("Failed to sync datastore. Error: %+v \n", syncDatastoreTaskInfo.Error)
	//	t.Fatalf("%+v", syncDatastoreTaskInfo.Error)
	//}
	//t.Logf("syncDatastore on %v successful\n", dsUrl)
	//
	//t.Logf("Calling syncDatastore on %v with fullsync...\n", dsUrl)
	//syncDatastoreTask, err = cnsClient.SyncDatastore(ctx, dsUrl, true)
	//if err != nil {
	//	t.Errorf("Failed to sync datastore %v with full sync. Error: %+v \n", dsUrl, err)
	//	t.Fatal(err)
	//}
	//syncDatastoreTaskInfo, err = GetTaskInfo(ctx, syncDatastoreTask)
	//if err != nil {
	//	t.Errorf("Failed to get sync datastore taskInfo with full sync. Error: %+v \n", err)
	//	t.Fatal(err)
	//}
	//if syncDatastoreTaskInfo.State != vim25types.TaskInfoStateSuccess {
	//	t.Errorf("Failed to sync datastore with full sync. Error: %+v \n", syncDatastoreTaskInfo.Error)
	//	t.Fatalf("%+v", syncDatastoreTaskInfo.Error)
	//}
	//t.Logf("syncDatastore on %v with full sync successful\n", dsUrl)
}

// isvSphereVersion70U3orAbove checks if specified version is 7.0 Update 3 or higher
// The method takes aboutInfo{} as input which contains details about
// VC version, build number and so on.
// If the version is 7.0 Update 3 or higher, the method returns true, else returns false
// along with appropriate errors during failure cases
func isvSphereVersion70U3orAbove(ctx context.Context, aboutInfo vim25types.AboutInfo) bool {
	items := strings.Split(aboutInfo.Version, ".")
	version := strings.Join(items[:], "")
	// Convert version string to string, Ex: "7.0.3" becomes 703, "7.0.3.1" becomes 703
	if len(version) >= 3 {
		vSphereVersionInt, err := strconv.Atoi(version[0:3])
		if err != nil {
			return false
		}
		// Check if the current vSphere version is 7.0.3 or higher
		if vSphereVersionInt >= VSphere70u3VersionInt {
			return true
		}
	}
	// For all other versions
	return false
}
