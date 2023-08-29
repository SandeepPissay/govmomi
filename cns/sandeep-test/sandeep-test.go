package main

import (
	"context"
	"fmt"
	"github.com/dougm/pretty"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/cns"
	cnstypes "github.com/vmware/govmomi/cns/types"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/soap"
	vim25types "github.com/vmware/govmomi/vim25/types"
	"os"
)

func main() {
	fmt.Println("Hello!")
	url := "https://Administrator@vsphere.local:TIYPDu.h-Gd5evv1@10.193.4.165/sdk"

	u, err := soap.ParseURL(url)
	if err != nil {
		ExitWithError("Failed to parse URL", err)
	}
	ctx := context.Background()
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		ExitWithError("Failed to create new client", err)
	}
	// UseServiceVersion sets soap.Client.Version to the current version of the service endpoint via /sdk/vsanServiceVersions.xml
	err = c.UseServiceVersion("vsan")
	if err != nil {
		ExitWithError("Failed to use service version", err)
	}
	cnsClient, err := cns.NewClient(ctx, c.Client)
	if err != nil {
		ExitWithError("Failed to create new CNS client", err)
	}
	finder := find.NewFinder(cnsClient.Vim25Client, false)
	dc, err := finder.Datacenter(ctx, "VSAN-DC")
	if err != nil {
		ExitWithError("Failed to find datacenter", err)
	}
	fmt.Println(dc)

	volId := createVolume(ctx, cnsClient, finder, dc)
	fmt.Printf("VolumeId: %+v\n", volId)
	snapshotId := createVolumeSnapshot(ctx, cnsClient, volId)
	fmt.Printf("snapshotId: %+v\n", snapshotId)
	createVolumeFromSnapshot(ctx, cnsClient, finder, dc, snapshotId, volId)
	deleteSnapshot(ctx, cnsClient, volId, snapshotId)
	deleteVolume(ctx, cnsClient, volId)
}

func deleteSnapshot(ctx context.Context, cnsClient *cns.Client, volumeId string, snapshotId string) {
	// Test DeleteSnapshot API
	// Construct the CNS SnapshotDeleteSpec list
	var cnsSnapshotDeleteSpecList []cnstypes.CnsSnapshotDeleteSpec
	cnsSnapshotDeleteSpec := cnstypes.CnsSnapshotDeleteSpec{
		VolumeId: cnstypes.CnsVolumeId{
			Id: volumeId,
		},
		SnapshotId: cnstypes.CnsSnapshotId{
			Id: snapshotId,
		},
	}
	cnsSnapshotDeleteSpecList = append(cnsSnapshotDeleteSpecList, cnsSnapshotDeleteSpec)
	fmt.Printf("Deleting snapshot using the spec: %+v\n", pretty.Sprint(cnsSnapshotDeleteSpecList))
	deleteSnapshotsTask, err := cnsClient.DeleteSnapshots(ctx, cnsSnapshotDeleteSpecList)
	if err != nil {
		ExitWithError("Failed to get the task of DeleteSnapshots.", err)
	}
	deleteSnapshotsTaskInfo, err := cns.GetTaskInfo(ctx, deleteSnapshotsTask)
	if err != nil {
		ExitWithError("Failed to get the task info of DeleteSnapshots.", err)
	}

	deleteSnapshotsTaskResult, err := cns.GetTaskResult(ctx, deleteSnapshotsTaskInfo)
	if err != nil {
		ExitWithError("Failed to get the task result of DeleteSnapshots.", err)
	}

	deleteSnapshotsOperationRes := deleteSnapshotsTaskResult.GetCnsVolumeOperationResult()
	if deleteSnapshotsOperationRes.Fault != nil {
		ExitWithError("Failed to delete snapshots", deleteSnapshotsOperationRes.Fault)
	}

	snapshotDeleteResult := interface{}(deleteSnapshotsTaskResult).(*cnstypes.CnsSnapshotDeleteResult)
	fmt.Printf("DeleteSnapshots: Snapshot deleted successfully. volumeId: %q, snapshot id %q, opId: %q\n",
		volumeId, snapshotDeleteResult.SnapshotId, deleteSnapshotsTaskInfo.ActivationId)
}

func createVolumeSnapshot(ctx context.Context, cnsClient *cns.Client, volumeId string) string {
	// Test CreateSnapshot API
	// Construct the CNS SnapshotCreateSpec list
	desc := "example-vanilla-block-snapshot"
	var cnsSnapshotCreateSpecList []cnstypes.CnsSnapshotCreateSpec
	cnsSnapshotCreateSpec := cnstypes.CnsSnapshotCreateSpec{
		VolumeId: cnstypes.CnsVolumeId{
			Id: volumeId,
		},
		Description: desc,
	}
	cnsSnapshotCreateSpecList = append(cnsSnapshotCreateSpecList, cnsSnapshotCreateSpec)
	fmt.Printf("Creating snapshot using the spec: %+v\n", pretty.Sprint(cnsSnapshotCreateSpecList))
	createSnapshotsTask, err := cnsClient.CreateSnapshots(ctx, cnsSnapshotCreateSpecList)
	if err != nil {
		ExitWithError("Failed to get the task of CreateSnapshot.", err)
	}
	createSnapshotsTaskInfo, err := cns.GetTaskInfo(ctx, createSnapshotsTask)
	if err != nil {
		ExitWithError("Failed to get the task info of CreateSnapshots.", err)
	}
	createSnapshotsTaskResult, err := cns.GetTaskResult(ctx, createSnapshotsTaskInfo)
	if err != nil {
		ExitWithError("Failed to get the task result of CreateSnapshots.", err)
	}
	createSnapshotsOperationRes := createSnapshotsTaskResult.GetCnsVolumeOperationResult()
	if createSnapshotsOperationRes.Fault != nil {
		ExitWithError("Failed to create snapshots", createSnapshotsOperationRes.Fault)
	}

	snapshotCreateResult := interface{}(createSnapshotsTaskResult).(*cnstypes.CnsSnapshotCreateResult)
	snapshotId := snapshotCreateResult.Snapshot.SnapshotId.Id
	snapshotCreateTime := snapshotCreateResult.Snapshot.CreateTime
	fmt.Printf("CreateSnapshots: Snapshot created successfully. volumeId: %q, snapshot id %q, time stamp %+v,"+
		" opId: %q\n", volumeId, snapshotId, snapshotCreateTime, createSnapshotsTaskInfo.ActivationId)
	return snapshotId
}

func createVolumeFromSnapshot(ctx context.Context, cnsClient *cns.Client, finder *find.Finder, dc *object.Datacenter,
	snapshotId string, volId string) {
	// Test CreateVolumeFromSnapshot functionality by calling CreateVolume with VolumeSource set
	// Query Volume for capacity
	var queryVolumeIDList []cnstypes.CnsVolumeId
	var queryFilter cnstypes.CnsQueryFilter
	queryVolumeIDList = append(queryVolumeIDList, cnstypes.CnsVolumeId{Id: volId})
	queryFilter.VolumeIds = queryVolumeIDList
	fmt.Printf("CreateVolumeFromSnapshot: calling QueryVolume using queryFilter: %+v\n", pretty.Sprint(queryFilter))
	queryResult, err := cnsClient.QueryVolume(ctx, queryFilter)
	if err != nil {
		ExitWithError("Failed to query volume.", err)
	}
	var snapshotSize int64
	if len(queryResult.Volumes) > 0 {
		snapshotSize = queryResult.Volumes[0].BackingObjectDetails.GetCnsBackingObjectDetails().CapacityInMb
	} else {
		msg := fmt.Sprintf("failed to get the snapshot size by querying volume: %q", volId)
		ExitWithError(msg, volId)
	}
	fmt.Printf("CreateVolumeFromSnapshot: Successfully Queried Volumes. queryResult: %+v\n", pretty.Sprint(queryResult))

	finder.SetDatacenter(dc)
	ds, err := finder.Datastore(ctx, "vsanDatastore")
	if err != nil {
		ExitWithError("Failed to find datastore", err)
	}

	var dsList []vim25types.ManagedObjectReference
	dsList = append(dsList, ds.Reference())

	containerCluster := cnstypes.CnsContainerCluster{
		ClusterType:         string(cnstypes.CnsClusterTypeKubernetes),
		ClusterId:           "demo-cluster-id",
		VSphereUser:         "Administrator@vsphere.local",
		ClusterFlavor:       string(cnstypes.CnsClusterFlavorVanilla),
		ClusterDistribution: "OpenShift",
	}

	// Construct the CNS VolumeCreateSpec list
	cnsCreateVolumeFromSnapshotCreateSpec := cnstypes.CnsVolumeCreateSpec{
		Name:       "pvc-901e87eb-c2bd-11e9-806f-005056a0c9a0-create-from-snapshot",
		VolumeType: string(cnstypes.CnsVolumeTypeBlock),
		Datastores: dsList,
		Metadata: cnstypes.CnsVolumeMetadata{
			ContainerCluster: containerCluster,
		},
		BackingObjectDetails: &cnstypes.CnsBlockBackingDetails{
			CnsBackingObjectDetails: cnstypes.CnsBackingObjectDetails{
				CapacityInMb: snapshotSize,
			},
		},
		VolumeSource: &cnstypes.CnsSnapshotVolumeSource{
			VolumeId: cnstypes.CnsVolumeId{
				Id: volId,
			},
			SnapshotId: cnstypes.CnsSnapshotId{
				Id: snapshotId,
			},
			LinkedClone: true,
		},
	}
	var cnsCreateVolumeFromSnapshotCreateSpecList []cnstypes.CnsVolumeCreateSpec
	cnsCreateVolumeFromSnapshotCreateSpecList = append(cnsCreateVolumeFromSnapshotCreateSpecList, cnsCreateVolumeFromSnapshotCreateSpec)
	fmt.Printf("Creating volume from snapshot using the spec: %+v", pretty.Sprint(cnsCreateVolumeFromSnapshotCreateSpec))
	createVolumeFromSnapshotTask, err := cnsClient.CreateVolume(ctx, cnsCreateVolumeFromSnapshotCreateSpecList)
	if err != nil {
		ExitWithError("Failed to create volume from snapshot.", err)
	}
	createVolumeFromSnapshotTaskInfo, err := cns.GetTaskInfo(ctx, createVolumeFromSnapshotTask)
	if err != nil {
		ExitWithError("Failed to get task info", err)
	}
	createVolumeFromSnapshotTaskResult, err := cns.GetTaskResult(ctx, createVolumeFromSnapshotTaskInfo)
	if err != nil {
		ExitWithError("Failed to create volume from snapshot.", err)
	}
	if createVolumeFromSnapshotTaskResult == nil {
		ExitWithError("Empty create task results", nil)
	}
	createVolumeFromSnapshotOperationRes := createVolumeFromSnapshotTaskResult.GetCnsVolumeOperationResult()
	if createVolumeFromSnapshotOperationRes.Fault != nil {
		fmt.Printf("Failed to create volume from snapshot: fault=%+v\n",
			createVolumeFromSnapshotOperationRes.Fault)
	}
	createVolumeFromSnapshotVolumeId := createVolumeFromSnapshotOperationRes.VolumeId.Id
	createVolumeFromSnapshotResult := (createVolumeFromSnapshotTaskResult).(*cnstypes.CnsVolumeCreateResult)
	fmt.Printf("createVolumeFromSnapshotResult %+v\n", createVolumeFromSnapshotResult)
	fmt.Printf("Volume created from snapshot %s sucessfully. volumeId: %s", snapshotId, createVolumeFromSnapshotVolumeId)

	//  Clean up volume created from snapshot above
	var deleteVolumeFromSnapshotVolumeIDList []cnstypes.CnsVolumeId
	deleteVolumeFromSnapshotVolumeIDList = append(deleteVolumeFromSnapshotVolumeIDList,
		cnstypes.CnsVolumeId{Id: createVolumeFromSnapshotVolumeId})
	fmt.Printf("Deleting volume: %+v\n", deleteVolumeFromSnapshotVolumeIDList)
	deleteVolumeFromSnapshotTask, err := cnsClient.DeleteVolume(ctx, deleteVolumeFromSnapshotVolumeIDList, true)
	if err != nil {
		ExitWithError("Failed to delete volume.", err)
	}
	deleteVolumeFromSnapshotTaskInfo, err := cns.GetTaskInfo(ctx, deleteVolumeFromSnapshotTask)
	if err != nil {
		ExitWithError("Failed to delete volume.", err)
	}
	deleteVolumeFromSnapshotTaskResult, err := cns.GetTaskResult(ctx, deleteVolumeFromSnapshotTaskInfo)
	if err != nil {
		ExitWithError("Failed to delete volume", err)
	}
	if deleteVolumeFromSnapshotTaskResult == nil {
		ExitWithError("Empty delete task results", nil)
	}
	deleteVolumeFromSnapshotOperationRes := deleteVolumeFromSnapshotTaskResult.GetCnsVolumeOperationResult()
	if deleteVolumeFromSnapshotOperationRes.Fault != nil {
		ExitWithError("Failed to delete volume", deleteVolumeFromSnapshotOperationRes.Fault)
	}
	fmt.Printf("Volume: %q deleted sucessfully\n", createVolumeFromSnapshotVolumeId)
}

func deleteVolume(ctx context.Context, cnsClient *cns.Client, volId string) {
	var volumeIDList []cnstypes.CnsVolumeId
	volumeIDList = append(volumeIDList, cnstypes.CnsVolumeId{Id: volId})

	deleteTask, err := cnsClient.DeleteVolume(ctx, volumeIDList, true)
	if err != nil {
		ExitWithError("Failed to delete volume", err)
	}
	deleteTaskInfo, err := cns.GetTaskInfo(ctx, deleteTask)
	if err != nil {
		ExitWithError("Failed to delget task info", err)
	}
	deleteTaskResult, err := cns.GetTaskResult(ctx, deleteTaskInfo)
	if err != nil {
		ExitWithError("Failed to get task result", err)
	}
	if deleteTaskResult == nil {
		ExitWithError("Empty delete task results", err)
	}
	deleteVolumeOperationRes := deleteTaskResult.GetCnsVolumeOperationResult()
	if deleteVolumeOperationRes.Fault != nil {
		ExitWithError("deleteVolumeOperationRes.Fault is not nil", deleteVolumeOperationRes.Fault)
	}
	fmt.Printf("volume:%q deleted sucessfully\n", volId)
}

func ExitWithError(msg string, in interface{}) {
	fmt.Printf("%s : %+v\n", msg, in)
	os.Exit(1)
}

func createVolume(ctx context.Context, cnsClient *cns.Client, finder *find.Finder, dc *object.Datacenter) string {
	finder.SetDatacenter(dc)
	ds, err := finder.Datastore(ctx, "vsanDatastore")
	if err != nil {
		ExitWithError("Failed to find datastore", err)
	}

	var dsList []vim25types.ManagedObjectReference
	dsList = append(dsList, ds.Reference())

	containerCluster := cnstypes.CnsContainerCluster{
		ClusterType:         string(cnstypes.CnsClusterTypeKubernetes),
		ClusterId:           "demo-cluster-id",
		VSphereUser:         "Administrator@vsphere.local",
		ClusterFlavor:       string(cnstypes.CnsClusterFlavorVanilla),
		ClusterDistribution: "OpenShift",
	}

	var cnsVolumeCreateSpecList []cnstypes.CnsVolumeCreateSpec
	cnsVolumeCreateSpec := cnstypes.CnsVolumeCreateSpec{
		Name:       "sandeep-1",
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
	}
	cnsVolumeCreateSpecList = append(cnsVolumeCreateSpecList, cnsVolumeCreateSpec)
	fmt.Printf("Creating volume using the spec: %+v\n", pretty.Sprint(cnsVolumeCreateSpec))
	createTask, err := cnsClient.CreateVolume(ctx, cnsVolumeCreateSpecList)
	if err != nil {
		ExitWithError("Failed to create volume", err)
	}
	createTaskInfo, err := cns.GetTaskInfo(ctx, createTask)
	if err != nil {
		ExitWithError("Failed to get task info", err)
	}
	createTaskResult, err := cns.GetTaskResult(ctx, createTaskInfo)
	if err != nil {
		ExitWithError("Failed to task result", err)
	}
	if createTaskResult == nil {
		ExitWithError("Empty create volume result", nil)
	}

	createVolumeOperationRes := createTaskResult.GetCnsVolumeOperationResult()
	if createVolumeOperationRes.Fault != nil {
		ExitWithError("CreateTaskResult has a fault", createVolumeOperationRes.Fault)
	}
	volumeId := createVolumeOperationRes.VolumeId.Id
	volumeCreateResult := (createTaskResult).(*cnstypes.CnsVolumeCreateResult)
	fmt.Printf("volumeCreateResult %+v", volumeCreateResult)
	fmt.Printf("Volume created sucessfully. volumeId: %s", volumeId)
	return volumeId
}
