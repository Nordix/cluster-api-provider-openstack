/*
Copyright 2024 The Kubernetes Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha6

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
	v1alpha6 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"
)

// OpenStackMachineLister helps list OpenStackMachines.
// All objects returned here must be treated as read-only.
type OpenStackMachineLister interface {
	// List lists all OpenStackMachines in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha6.OpenStackMachine, err error)
	// OpenStackMachines returns an object that can list and get OpenStackMachines.
	OpenStackMachines(namespace string) OpenStackMachineNamespaceLister
	OpenStackMachineListerExpansion
}

// openStackMachineLister implements the OpenStackMachineLister interface.
type openStackMachineLister struct {
	listers.ResourceIndexer[*v1alpha6.OpenStackMachine]
}

// NewOpenStackMachineLister returns a new OpenStackMachineLister.
func NewOpenStackMachineLister(indexer cache.Indexer) OpenStackMachineLister {
	return &openStackMachineLister{listers.New[*v1alpha6.OpenStackMachine](indexer, v1alpha6.Resource("openstackmachine"))}
}

// OpenStackMachines returns an object that can list and get OpenStackMachines.
func (s *openStackMachineLister) OpenStackMachines(namespace string) OpenStackMachineNamespaceLister {
	return openStackMachineNamespaceLister{listers.NewNamespaced[*v1alpha6.OpenStackMachine](s.ResourceIndexer, namespace)}
}

// OpenStackMachineNamespaceLister helps list and get OpenStackMachines.
// All objects returned here must be treated as read-only.
type OpenStackMachineNamespaceLister interface {
	// List lists all OpenStackMachines in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha6.OpenStackMachine, err error)
	// Get retrieves the OpenStackMachine from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha6.OpenStackMachine, error)
	OpenStackMachineNamespaceListerExpansion
}

// openStackMachineNamespaceLister implements the OpenStackMachineNamespaceLister
// interface.
type openStackMachineNamespaceLister struct {
	listers.ResourceIndexer[*v1alpha6.OpenStackMachine]
}