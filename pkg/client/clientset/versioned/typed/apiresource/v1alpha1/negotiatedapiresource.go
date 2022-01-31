/*
Copyright The KCP Authors.

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
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"

	v1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apiresource/v1alpha1"
	scheme "github.com/kcp-dev/kcp/pkg/client/clientset/versioned/scheme"
)

// NegotiatedAPIResourcesGetter has a method to return a NegotiatedAPIResourceInterface.
// A group's client should implement this interface.
type NegotiatedAPIResourcesGetter interface {
	NegotiatedAPIResources() NegotiatedAPIResourceInterface
}

// NegotiatedAPIResourceInterface has methods to work with NegotiatedAPIResource resources.
type NegotiatedAPIResourceInterface interface {
	Create(ctx context.Context, negotiatedAPIResource *v1alpha1.NegotiatedAPIResource, opts v1.CreateOptions) (*v1alpha1.NegotiatedAPIResource, error)
	Update(ctx context.Context, negotiatedAPIResource *v1alpha1.NegotiatedAPIResource, opts v1.UpdateOptions) (*v1alpha1.NegotiatedAPIResource, error)
	UpdateStatus(ctx context.Context, negotiatedAPIResource *v1alpha1.NegotiatedAPIResource, opts v1.UpdateOptions) (*v1alpha1.NegotiatedAPIResource, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.NegotiatedAPIResource, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NegotiatedAPIResourceList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NegotiatedAPIResource, err error)
	NegotiatedAPIResourceExpansion
}

// negotiatedAPIResources implements NegotiatedAPIResourceInterface
type negotiatedAPIResources struct {
	client  rest.Interface
	cluster string
}

// newNegotiatedAPIResources returns a NegotiatedAPIResources
func newNegotiatedAPIResources(c *ApiresourceV1alpha1Client) *negotiatedAPIResources {
	return &negotiatedAPIResources{
		client:  c.RESTClient(),
		cluster: c.cluster,
	}
}

// Get takes name of the negotiatedAPIResource, and returns the corresponding negotiatedAPIResource object, and an error if there is any.
func (c *negotiatedAPIResources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NegotiatedAPIResource, err error) {
	result = &v1alpha1.NegotiatedAPIResource{}
	err = c.client.Get().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NegotiatedAPIResources that match those selectors.
func (c *negotiatedAPIResources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NegotiatedAPIResourceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.NegotiatedAPIResourceList{}
	err = c.client.Get().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested negotiatedAPIResources.
func (c *negotiatedAPIResources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a negotiatedAPIResource and creates it.  Returns the server's representation of the negotiatedAPIResource, and an error, if there is any.
func (c *negotiatedAPIResources) Create(ctx context.Context, negotiatedAPIResource *v1alpha1.NegotiatedAPIResource, opts v1.CreateOptions) (result *v1alpha1.NegotiatedAPIResource, err error) {
	result = &v1alpha1.NegotiatedAPIResource{}
	err = c.client.Post().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(negotiatedAPIResource).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a negotiatedAPIResource and updates it. Returns the server's representation of the negotiatedAPIResource, and an error, if there is any.
func (c *negotiatedAPIResources) Update(ctx context.Context, negotiatedAPIResource *v1alpha1.NegotiatedAPIResource, opts v1.UpdateOptions) (result *v1alpha1.NegotiatedAPIResource, err error) {
	result = &v1alpha1.NegotiatedAPIResource{}
	err = c.client.Put().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		Name(negotiatedAPIResource.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(negotiatedAPIResource).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *negotiatedAPIResources) UpdateStatus(ctx context.Context, negotiatedAPIResource *v1alpha1.NegotiatedAPIResource, opts v1.UpdateOptions) (result *v1alpha1.NegotiatedAPIResource, err error) {
	result = &v1alpha1.NegotiatedAPIResource{}
	err = c.client.Put().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		Name(negotiatedAPIResource.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(negotiatedAPIResource).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the negotiatedAPIResource and deletes it. Returns an error if one occurs.
func (c *negotiatedAPIResources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *negotiatedAPIResources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched negotiatedAPIResource.
func (c *negotiatedAPIResources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NegotiatedAPIResource, err error) {
	result = &v1alpha1.NegotiatedAPIResource{}
	err = c.client.Patch(pt).
		Cluster(c.cluster).
		Resource("negotiatedapiresources").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
