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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha6

// NetworkParamApplyConfiguration represents a declarative configuration of the NetworkParam type for use
// with apply.
type NetworkParamApplyConfiguration struct {
	UUID    *string                          `json:"uuid,omitempty"`
	FixedIP *string                          `json:"fixedIP,omitempty"`
	Filter  *NetworkFilterApplyConfiguration `json:"filter,omitempty"`
	Subnets []SubnetParamApplyConfiguration  `json:"subnets,omitempty"`
}

// NetworkParamApplyConfiguration constructs a declarative configuration of the NetworkParam type for use with
// apply.
func NetworkParam() *NetworkParamApplyConfiguration {
	return &NetworkParamApplyConfiguration{}
}

// WithUUID sets the UUID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UUID field is set to the value of the last call.
func (b *NetworkParamApplyConfiguration) WithUUID(value string) *NetworkParamApplyConfiguration {
	b.UUID = &value
	return b
}

// WithFixedIP sets the FixedIP field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FixedIP field is set to the value of the last call.
func (b *NetworkParamApplyConfiguration) WithFixedIP(value string) *NetworkParamApplyConfiguration {
	b.FixedIP = &value
	return b
}

// WithFilter sets the Filter field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Filter field is set to the value of the last call.
func (b *NetworkParamApplyConfiguration) WithFilter(value *NetworkFilterApplyConfiguration) *NetworkParamApplyConfiguration {
	b.Filter = value
	return b
}

// WithSubnets adds the given value to the Subnets field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Subnets field.
func (b *NetworkParamApplyConfiguration) WithSubnets(values ...*SubnetParamApplyConfiguration) *NetworkParamApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSubnets")
		}
		b.Subnets = append(b.Subnets, *values[i])
	}
	return b
}
