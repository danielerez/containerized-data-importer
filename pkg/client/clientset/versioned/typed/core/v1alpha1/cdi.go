/*
Copyright 2018 The CDI Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	scheme "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned/scheme"
)

// CDIsGetter has a method to return a CDIInterface.
// A group's client should implement this interface.
type CDIsGetter interface {
	CDIs() CDIInterface
}

// CDIInterface has methods to work with CDI resources.
type CDIInterface interface {
	Create(*v1alpha1.CDI) (*v1alpha1.CDI, error)
	Update(*v1alpha1.CDI) (*v1alpha1.CDI, error)
	UpdateStatus(*v1alpha1.CDI) (*v1alpha1.CDI, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.CDI, error)
	List(opts v1.ListOptions) (*v1alpha1.CDIList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CDI, err error)
	CDIExpansion
}

// cDIs implements CDIInterface
type cDIs struct {
	client rest.Interface
}

// newCDIs returns a CDIs
func newCDIs(c *CdiV1alpha1Client) *cDIs {
	return &cDIs{
		client: c.RESTClient(),
	}
}

// Get takes name of the cDI, and returns the corresponding cDI object, and an error if there is any.
func (c *cDIs) Get(name string, options v1.GetOptions) (result *v1alpha1.CDI, err error) {
	result = &v1alpha1.CDI{}
	err = c.client.Get().
		Resource("cdis").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CDIs that match those selectors.
func (c *cDIs) List(opts v1.ListOptions) (result *v1alpha1.CDIList, err error) {
	result = &v1alpha1.CDIList{}
	err = c.client.Get().
		Resource("cdis").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested cDIs.
func (c *cDIs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("cdis").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a cDI and creates it.  Returns the server's representation of the cDI, and an error, if there is any.
func (c *cDIs) Create(cDI *v1alpha1.CDI) (result *v1alpha1.CDI, err error) {
	result = &v1alpha1.CDI{}
	err = c.client.Post().
		Resource("cdis").
		Body(cDI).
		Do().
		Into(result)
	return
}

// Update takes the representation of a cDI and updates it. Returns the server's representation of the cDI, and an error, if there is any.
func (c *cDIs) Update(cDI *v1alpha1.CDI) (result *v1alpha1.CDI, err error) {
	result = &v1alpha1.CDI{}
	err = c.client.Put().
		Resource("cdis").
		Name(cDI.Name).
		Body(cDI).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *cDIs) UpdateStatus(cDI *v1alpha1.CDI) (result *v1alpha1.CDI, err error) {
	result = &v1alpha1.CDI{}
	err = c.client.Put().
		Resource("cdis").
		Name(cDI.Name).
		SubResource("status").
		Body(cDI).
		Do().
		Into(result)
	return
}

// Delete takes name of the cDI and deletes it. Returns an error if one occurs.
func (c *cDIs) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("cdis").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *cDIs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("cdis").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched cDI.
func (c *cDIs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CDI, err error) {
	result = &v1alpha1.CDI{}
	err = c.client.Patch(pt).
		Resource("cdis").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
