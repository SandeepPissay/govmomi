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

	"github.com/dougm/pretty"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/debug"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/vmware/govmomi"
	cnstypes "github.com/vmware/govmomi/cns/types"
	vim25types "github.com/vmware/govmomi/vim25/types"
)

const VSphere70u3VersionInt = 703
const VSphere80u3VersionInt = 803

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

	var dsList []vim25types.ManagedObjectReference
	dsList = append(dsList, ds.Reference())

	var containerClusterArray []cnstypes.CnsContainerCluster
	containerCluster := cnstypes.CnsContainerCluster{
		ClusterType:         string(cnstypes.CnsClusterTypeKubernetes),
		ClusterId:           "demo-cluster-id",
		VSphereUser:         "Administrator@vsphere.local",
		ClusterFlavor:       string(cnstypes.CnsClusterFlavorVanilla),
		ClusterDistribution: "OpenShift",
	}
	containerClusterArray = append(containerClusterArray, containerCluster)

	// Test CreateVolume API
	var cnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	cnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
		Name:       "test1",
		VolumeType: string(cnstypes.CnsVolumeTypeBlock),
		Datastores: dsList,
		Metadata: cnstypes.CnsVolumeMetadata{
			ContainerCluster: containerCluster,
		},
		BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
			CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
				CapacityInMb: 5120,
			},
		},
		Profile: []vim25types.BaseVirtualMachineProfileSpec{
			&vim25types.VirtualMachineDefinedProfileSpec{ProfileId: "e3a88e6c-5741-4e02-8b3a-1d8663c6496e"},
		},
		//CreateSpec: &cnstypes.CnsBlockCreateSpec{  // TODO: This seems to be failing in CNS with some XML parsing error!
		//	CryptoSpec: &vim25types.CryptoSpecEncrypt{
		//		CryptoKeyId: vim25types.CryptoKeyId{
		//			KeyId: "6",
		//			ProviderId: &vim25types.KeyProviderId{
		//				Id: "kmip1",
		//			},
		//		},
		//	},
		//},
	}
	cnsVolumeCreateSpecList = append(cnsVolumeCreateSpecList, cnsVolumeCreateSpec)
	t.Logf("Creating volume using the spec: %+v", pretty.Sprint(cnsVolumeCreateSpec))
	createTask, err := cnsClient.CreateVolume(ctx, cnsVolumeCreateSpecList)
	if err != nil {
		t.Errorf("Failed to create volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	createTaskInfo, err := GetTaskInfo(ctx, createTask)
	if err != nil {
		t.Errorf("Failed to create volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	createTaskResult, err := GetTaskResult(ctx, createTaskInfo)
	if err != nil {
		t.Errorf("Failed to create volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	if createTaskResult == nil {
		t.Fatalf("Empty create task results")
		t.FailNow()
	}
	createVolumeOperationRes := createTaskResult.GetCnsVolumeOperationResult()
	if createVolumeOperationRes.Fault != nil {
		t.Fatalf("Failed to create volume: fault=%+v", createVolumeOperationRes.Fault)
	}
	volumeId := createVolumeOperationRes.VolumeId.Id
	volumeCreateResult := (createTaskResult).(*cnstypes.CnsVolumeCreateResult)
	t.Logf("volumeCreateResult %+v", volumeCreateResult)
	t.Logf("Volume created sucessfully. volumeId: %s", volumeId)

	// Test QueryVolume API
	var queryFilter cnstypes.CnsQueryFilter
	var volumeIDList []cnstypes.CnsVolumeId
	volumeIDList = append(volumeIDList, cnstypes.CnsVolumeId{Id: volumeId})
	queryFilter.VolumeIds = volumeIDList
	t.Logf("Calling QueryVolume using queryFilter: %+v", pretty.Sprint(queryFilter))
	queryResult, err := cnsClient.QueryVolume(ctx, queryFilter)
	if err != nil {
		t.Errorf("Failed to query volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	t.Logf("Successfully Queried Volumes. queryResult: %+v", pretty.Sprint(queryResult))
	t.Logf("Before recrypt - QueryVolumeInfo")
	QueryVolumeInfo(t, ctx, cnsClient, volumeIDList)

	// Create a VM to test Attach Volume API.
	virtualMachineConfigSpec := vim25types.VirtualMachineConfigSpec{
		Name:    "test-node-vm",
		GuestId: string(vim25types.VirtualMachineGuestOsIdentifierOtherGuest),
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
						},
					},
				},
			},
		},
		VmProfile: []vim25types.BaseVirtualMachineProfileSpec{
			&vim25types.VirtualMachineDefinedProfileSpec{ProfileId: "e3a88e6c-5741-4e02-8b3a-1d8663c6496e"},
		},
		Crypto: &vim25types.CryptoSpecEncrypt{
			CryptoKeyId: vim25types.CryptoKeyId{
				KeyId: "6",
				ProviderId: &vim25types.KeyProviderId{
					Id: "kmip1",
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

	nodeVM := object.NewVirtualMachine(cnsClient.vim25Client, vmRef.Reference())

	// Power ON the VM
	task, err = nodeVM.PowerOn(ctx)
	err = task.Wait(ctx)
	if err != nil {
		t.Fatalf("VM PowerOn task failed: %+v", err)
	}
	defer func() {
		// Power OFF the VM
		task, err = nodeVM.PowerOff(ctx)
		err = task.Wait(ctx)
		if err != nil {
			t.Fatalf("VM PowerOff task failed: %+v", err)
		}
		task, err = nodeVM.Destroy(ctx)
		if err != nil {
			t.Fatalf("VM Destroy task failed: %+v", err)
		}
	}()

	// Test AttachVolume API
	var cnsVolumeAttachSpecList []cnstypes.CnsVolumeAttachDetachSpec
	cnsVolumeAttachSpec := cnstypes.CnsVolumeAttachDetachSpec{
		VolumeId: cnstypes.CnsVolumeId{
			Id: volumeId,
		},
		Vm: nodeVM.Reference(),
	}
	cnsVolumeAttachSpecList = append(cnsVolumeAttachSpecList, cnsVolumeAttachSpec)
	t.Logf("Attaching volume using the spec: %+v", cnsVolumeAttachSpec)
	attachTask, err := cnsClient.AttachVolume(ctx, cnsVolumeAttachSpecList)
	if err != nil {
		t.Errorf("Failed to attach volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	attachTaskInfo, err := GetTaskInfo(ctx, attachTask)
	if err != nil {
		t.Errorf("Failed to attach volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	attachTaskResult, err := GetTaskResult(ctx, attachTaskInfo)
	if err != nil {
		t.Errorf("Failed to attach volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	if attachTaskResult == nil {
		t.Fatalf("Empty attach task results")
		t.FailNow()
	}
	attachVolumeOperationRes := attachTaskResult.GetCnsVolumeOperationResult()
	if attachVolumeOperationRes.Fault != nil {
		t.Fatalf("Failed to attach volume: fault=%+v", attachVolumeOperationRes.Fault)
	}
	diskUUID := interface{}(attachTaskResult).(*cnstypes.CnsVolumeAttachResult).DiskUUID
	t.Logf("Volume attached sucessfully. Disk UUID: %s", diskUUID)

	// Recrypt the attached FCD via VM API.
	t.Logf("Shallow Recryping the VM with a different KeyId...")
	devices, err := nodeVM.Device(ctx)
	if err != nil {
		t.Fatalf("error getting the devices. devices: %+v", devices)
	}
	var fcd *vim25types.VirtualDisk
	for _, device := range devices {
		switch md := device.(type) {
		case *vim25types.VirtualDisk:
			if md.VDiskId.Id == volumeId {
				fcd = md
			}
		default:
			continue
		}
	}

	spec := vim25types.VirtualMachineConfigSpec{}
	config := &vim25types.VirtualDeviceConfigSpec{
		Device: fcd,
		Backing: &vim25types.VirtualDeviceConfigSpecBackingSpec{
			Crypto: &vim25types.CryptoSpecShallowRecrypt{
				NewKeyId: vim25types.CryptoKeyId{
					KeyId: "7",
					ProviderId: &vim25types.KeyProviderId{
						Id: "kmip1",
					},
				},
			},
		},
		Profile: []vim25types.BaseVirtualMachineProfileSpec{
			&vim25types.VirtualMachineDefinedProfileSpec{ProfileId: "e3a88e6c-5741-4e02-8b3a-1d8663c6496e"},
		},
		Operation: vim25types.VirtualDeviceConfigSpecOperationEdit,
	}
	config.FileOperation = ""
	spec.DeviceChange = append(spec.DeviceChange, config)
	task, err = nodeVM.Reconfigure(ctx, spec)
	if err != nil {
		t.Fatalf("Reconfigure task submission failed: %+v", err)
	}
	err = task.Wait(ctx)
	if err != nil {
		t.Fatalf("Reconfigure task failed: %+v", err)
	}

	t.Logf("Shallow Recryping the VM Successful.")

	t.Logf("After recrypt - QueryVolumeInfo")
	QueryVolumeInfo(t, ctx, cnsClient, volumeIDList)

	// Test DetachVolume API
	var cnsVolumeDetachSpecList []cnstypes.CnsVolumeAttachDetachSpec
	cnsVolumeDetachSpec := cnstypes.CnsVolumeAttachDetachSpec{
		VolumeId: cnstypes.CnsVolumeId{
			Id: volumeId,
		},
		Vm: nodeVM.Reference(),
	}
	cnsVolumeDetachSpecList = append(cnsVolumeDetachSpecList, cnsVolumeDetachSpec)
	t.Logf("Detaching volume using the spec: %+v", cnsVolumeDetachSpec)
	detachTask, err := cnsClient.DetachVolume(ctx, cnsVolumeDetachSpecList)
	if err != nil {
		t.Errorf("Failed to detach volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	detachTaskInfo, err := GetTaskInfo(ctx, detachTask)
	if err != nil {
		t.Errorf("Failed to detach volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	detachTaskResult, err := GetTaskResult(ctx, detachTaskInfo)
	if err != nil {
		t.Errorf("Failed to detach volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	if detachTaskResult == nil {
		t.Fatalf("Empty detach task results")
		t.FailNow()
	}
	detachVolumeOperationRes := detachTaskResult.GetCnsVolumeOperationResult()
	if detachVolumeOperationRes.Fault != nil {
		t.Fatalf("Failed to detach volume: fault=%+v", detachVolumeOperationRes.Fault)
	}
	t.Logf("Volume detached sucessfully")

	// Test DeleteVolume API
	t.Logf("Deleting volume: %+v", volumeIDList)
	deleteTask, err := cnsClient.DeleteVolume(ctx, volumeIDList, true)
	if err != nil {
		t.Errorf("Failed to delete volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	deleteTaskInfo, err := GetTaskInfo(ctx, deleteTask)
	if err != nil {
		t.Errorf("Failed to delete volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	deleteTaskResult, err := GetTaskResult(ctx, deleteTaskInfo)
	if err != nil {
		t.Errorf("Failed to delete volume. Error: %+v \n", err)
		t.Fatal(err)
	}
	if deleteTaskResult == nil {
		t.Fatalf("Empty delete task results")
		t.FailNow()
	}
	deleteVolumeOperationRes := deleteTaskResult.GetCnsVolumeOperationResult()
	if deleteVolumeOperationRes.Fault != nil {
		t.Fatalf("Failed to delete volume: fault=%+v", deleteVolumeOperationRes.Fault)
	}
	t.Logf("Volume: %q deleted sucessfully", volumeId)
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

func QueryVolumeInfo(t *testing.T, ctx context.Context, cnsClient *Client, volumeIDList []cnstypes.CnsVolumeId) {
	if cnsClient.Version != ReleaseVSAN67u3 && cnsClient.Version != ReleaseVSAN70 {
		t.Logf("Calling QueryVolumeInfo using: %+v", pretty.Sprint(volumeIDList))
		queryVolumeInfoTask, err := cnsClient.QueryVolumeInfo(ctx, volumeIDList)
		if err != nil {
			t.Errorf("Failed to query volumes with QueryVolumeInfo. Error: %+v \n", err)
			t.Fatal(err)
		}
		queryVolumeInfoTaskInfo, err := GetTaskInfo(ctx, queryVolumeInfoTask)
		if err != nil {
			t.Errorf("Failed to query volumes with QueryVolumeInfo. Error: %+v \n", err)
			t.Fatal(err)
		}
		queryVolumeInfoTaskResults, err := GetTaskResultArray(ctx, queryVolumeInfoTaskInfo)
		if err != nil {
			t.Errorf("Failed to query volumes with QueryVolumeInfo. Error: %+v \n", err)
			t.Fatal(err)
		}
		if queryVolumeInfoTaskResults == nil {
			t.Fatalf("Empty queryVolumeInfoTaskResult")
			t.FailNow()
		}
		for _, queryVolumeInfoTaskResult := range queryVolumeInfoTaskResults {
			queryVolumeInfoOperationRes := queryVolumeInfoTaskResult.GetCnsVolumeOperationResult()
			if queryVolumeInfoOperationRes.Fault != nil {
				t.Fatalf("Failed to query volumes with QueryVolumeInfo. fault=%+v", queryVolumeInfoOperationRes.Fault)
			}
			t.Logf("Successfully Queried Volumes. queryVolumeInfoTaskResult: %+v", pretty.Sprint(queryVolumeInfoTaskResult))
			//volumeInfoResult := interface{}(queryVolumeInfoTaskResult).(*cnstypes.CnsQueryVolumeInfoResult)
			//cnsBlockVolumeInfo := interface{}(volumeInfoResult.VolumeInfo).(*cnstypes.CnsBlockVolumeInfo)
			//t.Logf("cnsBlockVolumeInfo = %+v", pretty.Sprint(cnsBlockVolumeInfo))
		}
	}

}

// isvSphereVersion80U3orAbove checks if specified version is 8.0 Update 3 or higher
// The method takes aboutInfo{} as input which contains details about
// VC version, build number and so on.
// If the version is 8.0 Update 3 or higher, the method returns true, else returns false
// along with appropriate errors during failure cases
func isvSphereVersion80U3orAbove(ctx context.Context, aboutInfo vim25types.AboutInfo) bool {
	items := strings.Split(aboutInfo.Version, ".")
	version := strings.Join(items[:], "")
	// Convert version string to string, Ex: "8.0.3" becomes 803, "8.0.3.1" becomes 703
	if len(version) >= 3 {
		vSphereVersionInt, err := strconv.Atoi(version[0:3])
		if err != nil {
			return false
		}
		// Check if the current vSphere version is 8.0.3 or higher
		if vSphereVersionInt >= VSphere80u3VersionInt {
			return true
		}
	}
	// For all other versions
	return false
}
