/*
Copyright (c) 2017 VMware, Inc. All Rights Reserved.

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

package simulator_test

import (
	"context"
	"fmt"
	"log"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// BusyVM changes the behavior of simulator.VirtualMachine
type BusyVM struct {
	*simulator.VirtualMachine
}

// Override PowerOffVMTask to inject a fault
func (vm *BusyVM) PowerOffVMTask(req *types.PowerOffVM_Task) soap.HasFault {
	task := simulator.CreateTask(vm.Self, "powerOff", func(*simulator.Task) (types.AnyType, types.BaseMethodFault) {
		return nil, &types.TaskInProgress{}
	})
	task.Run() // TODO:
	return &methods.PowerOffVM_TaskBody{Res: &types.PowerOffVM_TaskResponse{
		Returnval: task.Self,
	}}
}

// Example of extending the simulator to change behavior
func Example() {
	ctx := context.Background()
	model := simulator.ESX()

	defer model.Remove()
	_ = model.Create()

	s := model.Service.NewServer()
	defer s.Close()

	c, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		log.Fatal(err)
	}

	// Any VM will do
	vm := simulator.Map.Any("VirtualMachine").(*simulator.VirtualMachine)

	// Wrap existing vm object
	simulator.Map.Put(&BusyVM{vm})

	task, err := object.NewVirtualMachine(c.Client, vm.Reference()).PowerOff(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = task.Wait(ctx)
	if err == nil {
		log.Fatal("expected error")
	}

	fmt.Println(vm.Runtime.PowerState)
	// Output: poweredOn
}
