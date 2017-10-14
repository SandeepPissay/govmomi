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
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/simulator"
)

// Example boilerplate for starting a simulator initialized with a vCenter model.
func ExampleVPX() {
	ctx := context.Background()

	// vCenter model + initial set of objects (cluster, hosts, VMs, network, datastore, etc)
	model := simulator.VPX()

	defer model.Remove()
	err := model.Create()
	if err != nil {
		log.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	c, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(c.Client.ServiceContent.About.ApiType)
	// Output: VirtualCenter
}

func ExampleEmptyVPX() {
	ctx := context.Background()

	// vCenter model without a Datacenter, similar to a fresh install of vCenter
	model := simulator.VPX()
	model.Datacenter = 0

	defer model.Remove()
	err := model.Create()
	if err != nil {
		log.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	c, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		log.Fatal(err)
	}

	dc, _ := find.NewFinder(c.Client, false).DatacenterList(ctx, "*")
	fmt.Printf("%s with %d Datacenter", c.Client.ServiceContent.About.ApiType, len(dc))
	// Output: VirtualCenter with 0 Datacenter
}

// Example boilerplate for starting a simulator initialized with an ESX model.
func ExampleESX() {
	ctx := context.Background()

	// ESXi model + initial set of objects (VMs, network, datastore)
	model := simulator.ESX()

	defer model.Remove()
	err := model.Create()
	if err != nil {
		log.Fatal(err)
	}

	s := model.Service.NewServer()
	defer s.Close()

	c, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(c.Client.ServiceContent.About.ApiType)
	// Output: HostAgent
}
