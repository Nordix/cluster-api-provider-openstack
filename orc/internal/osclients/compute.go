/*
Copyright 2021 The Kubernetes Authors.

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

package osclients

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	uflavors "github.com/gophercloud/utils/v2/openstack/compute/v2/flavors"
)

/*
NovaMinimumMicroversion is the minimum Nova microversion supported by CAPO
2.60 corresponds to OpenStack Queens

For the canonical description of Nova microversions, see
https://docs.openstack.org/nova/latest/reference/api-microversion-history.html

CAPO uses server tags, which were added in microversion 2.52.
CAPO supports multiattach volume types, which were added in microversion 2.60.
*/
const NovaMinimumMicroversion = "2.60"

type ComputeClient interface {
	ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error)

	GetFlavorFromName(flavor string) (*flavors.Flavor, error)
	CreateServer(createOpts servers.CreateOptsBuilder, schedulerHints servers.SchedulerHintOptsBuilder) (*servers.Server, error)
	DeleteServer(serverID string) error
	GetServer(serverID string) (*servers.Server, error)
	ListServers(listOpts servers.ListOptsBuilder) ([]servers.Server, error)

	ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error)
	DeleteAttachedInterface(serverID, portID string) error

	ListServerGroups() ([]servergroups.ServerGroup, error)
}

type computeClient struct{ client *gophercloud.ServiceClient }

// NewComputeClient returns a new compute client.
func NewComputeClient(providerClient *gophercloud.ProviderClient, providerClientOpts *clientconfig.ClientOpts) (ComputeClient, error) {
	compute, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region:       providerClientOpts.RegionName,
		Availability: clientconfig.GetEndpointType(providerClientOpts.EndpointType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service client: %v", err)
	}
	compute.Microversion = NovaMinimumMicroversion

	return &computeClient{compute}, nil
}

func (c computeClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	allPages, err := availabilityzones.List(c.client).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return availabilityzones.ExtractAvailabilityZones(allPages)
}

func (c computeClient) GetFlavorFromName(flavor string) (*flavors.Flavor, error) {
	flavorID, err := uflavors.IDFromName(context.TODO(), c.client, flavor)
	if err != nil {
		return nil, err
	}

	return flavors.Get(context.TODO(), c.client, flavorID).Extract()
}

func (c computeClient) CreateServer(createOpts servers.CreateOptsBuilder, schedulerHints servers.SchedulerHintOptsBuilder) (*servers.Server, error) {
	return servers.Create(context.TODO(), c.client, createOpts, schedulerHints).Extract()
}

func (c computeClient) DeleteServer(serverID string) error {
	return servers.Delete(context.TODO(), c.client, serverID).ExtractErr()
}

func (c computeClient) GetServer(serverID string) (*servers.Server, error) {
	var server servers.Server
	err := servers.Get(context.TODO(), c.client, serverID).ExtractInto(&server)
	if err != nil {
		return nil, err
	}
	return &server, nil
}

func (c computeClient) ListServers(listOpts servers.ListOptsBuilder) ([]servers.Server, error) {
	var serverList []servers.Server
	allPages, err := servers.List(c.client, listOpts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	err = servers.ExtractServersInto(allPages, &serverList)
	return serverList, err
}

func (c computeClient) ListAttachedInterfaces(serverID string) ([]attachinterfaces.Interface, error) {
	interfaces, err := attachinterfaces.List(c.client, serverID).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return attachinterfaces.ExtractInterfaces(interfaces)
}

func (c computeClient) DeleteAttachedInterface(serverID, portID string) error {
	return attachinterfaces.Delete(context.TODO(), c.client, serverID, portID).ExtractErr()
}

func (c computeClient) ListServerGroups() ([]servergroups.ServerGroup, error) {
	opts := servergroups.ListOpts{}
	allPages, err := servergroups.List(c.client, opts).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return servergroups.ExtractServerGroups(allPages)
}

type computeErrorClient struct{ error }

// NewComputeErrorClient returns a ComputeClient in which every method returns the given error.
func NewComputeErrorClient(e error) ComputeClient {
	return computeErrorClient{e}
}

func (e computeErrorClient) ListAvailabilityZones() ([]availabilityzones.AvailabilityZone, error) {
	return nil, e.error
}

func (e computeErrorClient) GetFlavorFromName(_ string) (*flavors.Flavor, error) {
	return nil, e.error
}

func (e computeErrorClient) CreateServer(_ servers.CreateOptsBuilder, _ servers.SchedulerHintOptsBuilder) (*servers.Server, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteServer(_ string) error {
	return e.error
}

func (e computeErrorClient) GetServer(_ string) (*servers.Server, error) {
	return nil, e.error
}

func (e computeErrorClient) ListServers(_ servers.ListOptsBuilder) ([]servers.Server, error) {
	return nil, e.error
}

func (e computeErrorClient) ListAttachedInterfaces(_ string) ([]attachinterfaces.Interface, error) {
	return nil, e.error
}

func (e computeErrorClient) DeleteAttachedInterface(_, _ string) error {
	return e.error
}

func (e computeErrorClient) ListServerGroups() ([]servergroups.ServerGroup, error) {
	return nil, e.error
}
