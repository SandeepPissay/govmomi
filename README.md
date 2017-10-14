[![Build Status](https://travis-ci.org/vmware/govmomi.png?branch=master)](https://travis-ci.org/vmware/govmomi)
[![Go Report Card](https://goreportcard.com/badge/github.com/vmware/govmomi)](https://goreportcard.com/report/github.com/vmware/govmomi)

# govmomi

A Go library for interacting with VMware vSphere APIs (ESXi and/or vCenter).

## Components

### vSphere client sdk - [vim25](https://godoc.org/github.com/vmware/govmomi/vim25)

In addition to the generated sdk code, govmomi provides higher level APIs on top, making complex (but common) tasks
simpler.  These include:

- [Session management](https://godoc.org/github.com/vmware/govmomi/session)

- [Inventory navigation](https://godoc.org/github.com/vmware/govmomi/find)

- [Performance metrics](https://godoc.org/github.com/vmware/govmomi/performance)

- [Events](https://godoc.org/github.com/vmware/govmomi/event)

- [Tasks](https://godoc.org/github.com/vmware/govmomi/task)

- [Logs](https://godoc.org/github.com/vmware/govmomi/object#DiagnosticLog)

- [Property collection](https://godoc.org/github.com/vmware/govmomi/property)

- [Virtual devices](https://godoc.org/github.com/vmware/govmomi/object#VirtualDeviceList)

- [License management](https://godoc.org/github.com/vmware/govmomi/license)

- [Firewall configuration](https://godoc.org/github.com/vmware/govmomi/object#HostFirewallRulesetList)

- [Guest operations](https://godoc.org/github.com/vmware/govmomi/guest)

- [OVF import/export](https://godoc.org/github.com/vmware/govmomi/ovf)

### vSphere CLI - [govc](./govc)

In general, govc serves a few purposes:

- User friendly CLI alternative to the GUI clients

- Automation

- Test harness for the sdk and gomvomi APIs

- Examples of how to use the sdk and gomvomi APIs

For more details, see: [USAGE](./govc/USAGE.md)

### vSphere simulator - [vcsim](./vcsim)

The [simulator](https://godoc.org/github.com/vmware/govmomi/simulator) package provides an SDK endpoint mock framework
for simulating API interaction with vCenter and ESXi.

### Guest tools framework - [toolbox](./toolbox)

The [toolbox](https://godoc.org/github.com/vmware/govmomi/toolbox) package is a lightweight, extensible framework for
implementing VMware guest tools functionality.

## Compatibility

This library is built for and tested against ESXi and vCenter 6.0 and 6.5.

It should work with versions 5.5 and 5.1, but neither are officially supported.

## Documentation

The APIs exposed by this library very closely follow the API described in the [VMware vSphere API Reference Documentation][apiref].
Refer to this document to become familiar with the upstream API.

The code in the `govmomi` package is a wrapper for the code that is generated from the vSphere API description.
It primarily provides convenience functions for working with the vSphere API.
See [godoc.org][godoc] for documentation.

[apiref]:http://pubs.vmware.com/vsphere-6-5/index.jsp#com.vmware.wssdk.apiref.doc/right-pane.html
[godoc]:http://godoc.org/github.com/vmware/govmomi

## Installation

```sh
go get -u github.com/vmware/govmomi
```

## Discussion

Contributors and users are encouraged to collaborate using GitHub issues and/or
[Slack](https://vmwarecode.slack.com/messages/govmomi).
Access to Slack requires a [VMware {code} membership](https://code.vmware.com/join/).

## Status

Changes to the API are subject to [semantic versioning](http://semver.org).

Refer to the [CHANGELOG](CHANGELOG.md) for version to version changes.

## Projects using govmomi

* [Docker Machine](https://github.com/docker/machine/tree/master/drivers/vmwarevsphere)

* [Docker InfraKit](https://github.com/docker/infrakit/tree/master/pkg/provider/vsphere)

* [Docker LinuxKit](https://github.com/linuxkit/linuxkit/tree/master/src/cmd/linuxkit)

* [Kubernetes](https://github.com/kubernetes/kubernetes/tree/master/pkg/cloudprovider/providers/vsphere)

* [Kubernetes kops](https://github.com/kubernetes/kops/tree/master/upup/pkg/fi/cloudup/vsphere)

* [Terraform](https://github.com/terraform-providers/terraform-provider-vsphere)

* [Packer](https://github.com/jetbrains-infra/packer-builder-vsphere)

* [VMware VIC Engine](https://github.com/vmware/vic)

* [Travis CI](https://github.com/travis-ci/jupiter-brain)

* [collectd-vsphere](https://github.com/travis-ci/collectd-vsphere)

* [Gru](https://github.com/dnaeon/gru)

* [Libretto](https://github.com/apcera/libretto/tree/master/virtualmachine/vsphere)

## Related projects

* [rbvmomi](https://github.com/vmware/rbvmomi)

* [pyvmomi](https://github.com/vmware/pyvmomi)

## License

govmomi is available under the [Apache 2 license](LICENSE).
