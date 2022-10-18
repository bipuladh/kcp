package upsync

import (
	"context"
	"fmt"

	"github.com/kcp-dev/kcp/pkg/syncer/shared"
	. "github.com/kcp-dev/kcp/tmc/pkg/logging"
	"github.com/kcp-dev/logicalcluster/v2"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func isUpstreamKey(key string) bool {
	return false
}

func (c *Controller) processUpstreamResource(ctx context.Context, gvr schema.GroupVersionResource, key string) error {
	logger := klog.FromContext(ctx)
	// If it has no namespace locator then it is a upsynced resource
	// For such resource how do we check the downstream resource is still present?
	// Get the upstream namespace from the key
	upstreamNamespace, upstreamName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error(err, "Invalid key")
		return nil
	}
	err = c.upstreamClient.Cluster(c.syncTargetWorkspace).Resource(gvr).Namespace(upstreamNamespace).Delete(ctx, upstreamName, metav1.DeleteOptions{})
	if err != nil {
		logger.Error(err, "Could not remove resource upstream")
		return nil
	}
	return nil
}

func (c *Controller) processDownstreamResource(ctx context.Context, gvr schema.GroupVersionResource, key string) error {
	// Two conditions
	// A new resource
	// Existing resource
	// For existing resource just update

	logger := klog.FromContext(ctx)
	downstreamNamespace, downstreamName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error(err, "Invalid key")
		return nil
	}

	logger = logger.WithValues(DownstreamNamespace, downstreamNamespace, DownstreamName, downstreamName)

	var upstreamLocator *shared.NamespaceLocator
	var locatorExists bool
	var obj interface{}
	// Namespace scoped resource
	if downstreamNamespace != "" {
		var nsObj runtime.Object
		if nsObj, err = c.downstreamNamespaceLister.Get(downstreamNamespace); err != nil {
			logger.Error(err, "Error getting downstream Namespace from downstream namespace lister", "ns", downstreamNamespace)
		}
		nsMeta, ok := nsObj.(metav1.Object)
		if !ok {
			logger.Info(fmt.Sprintf("Error: downstream ns expected to be metav1.Object got %T", nsObj))
		}

		upstreamLocator, locatorExists, err = shared.LocatorFromAnnotations(nsMeta.GetAnnotations())
		if err != nil {
			logger.Error(err, "Error ")
		}

		if !locatorExists || upstreamLocator == nil {
			return nil
		}
		obj, err = c.downstreamClient.Resource(gvr).Namespace(downstreamNamespace).Get(ctx, downstreamName, metav1.GetOptions{})
		if err != nil {
			logger.Error(err, "Could not find the resource downstream")
		}
	}
	if downstreamNamespace == "" {
		obj, err = c.downstreamClient.Resource(gvr).Get(ctx, downstreamName, metav1.GetOptions{})
		if err != nil {
			logger.Error(err, "Could not find the resource downstream")
		}
		objMeta, ok := obj.(metav1.Object)
		if !ok {
			logger.Info("Error: downstream cluster-wide res not of metav1 type")
		}
		if upstreamLocator, locatorExists, err = shared.LocatorFromAnnotations(objMeta.GetAnnotations()); err != nil {
			logger.Error(err, "Error decoding annotations on DS cluster-wide res")
		}

		if !locatorExists || upstreamLocator == nil {
			return nil
		}
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		// type mismatch
		logger.Info("Type mismatch of resource object")
		return nil
	}

	if upstreamLocator.SyncTarget.UID != c.syncTargetUID || upstreamLocator.SyncTarget.Workspace != c.syncTargetWorkspace.String() {
		return nil
	}

	upstreamNamespace := upstreamLocator.Namespace
	upstreamWorkspace := upstreamLocator.Workspace

	objectName := u.GetName()

	// Resource exists upstream
	upstreamResource, err := c.upstreamClient.Cluster(upstreamWorkspace).Resource(gvr).Namespace(upstreamNamespace).Get(ctx, objectName, metav1.GetOptions{})
	if k8serror.IsNotFound(err) {
		logger.Info("Creating resource upstream", upstreamNamespace, upstreamWorkspace, u)
		c.createResourceInUpstream(ctx, gvr, upstreamNamespace, upstreamWorkspace, u)
	}
	if !k8serror.IsNotFound(err) && err != nil {
		return err
	}
	if upstreamResource != nil {
		c.updateResourceInUpstream(ctx, gvr, upstreamResource.GetNamespace(), upstreamWorkspace, u)
	}

	return nil
}

func (c *Controller) process(ctx context.Context, gvr schema.GroupVersionResource, key string, isUpstream bool) error {

	// get the object
	// delete it upstream

	if isUpstream {
		return c.processUpstreamResource(ctx, gvr, key)
	} else {
		return c.processDownstreamResource(ctx, gvr, key)
	}
}

func (c *Controller) updateResourceInUpstream(ctx context.Context, gvr schema.GroupVersionResource, upstreamNS string, upstreamLogicalCluster logicalcluster.Name, obj *unstructured.Unstructured) error {
	// Make a deepcopy
	updatedObject := obj.DeepCopy()
	annotations := updatedObject.GetAnnotations()
	delete(annotations, shared.NamespaceLocatorAnnotation)
	if (len(annotations)) == 0 {
		updatedObject.SetAnnotations(nil)
	} else {
		updatedObject.SetAnnotations(annotations)
	}
	// Create the resource
	if _, err := c.upstreamClient.Cluster(upstreamLogicalCluster).Resource(gvr).Namespace(upstreamNS).Update(ctx, updatedObject, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (c *Controller) createResourceInUpstream(ctx context.Context, gvr schema.GroupVersionResource, upstreamNS string, upstreamLogicalCluster logicalcluster.Name, downstreamObj *unstructured.Unstructured) error {
	// Make a deepcopy
	downstreamObject := downstreamObj.DeepCopy()
	annotations := downstreamObject.GetAnnotations()
	delete(annotations, shared.NamespaceLocatorAnnotation)
	if (len(annotations)) == 0 {
		downstreamObject.SetAnnotations(nil)
	} else {
		downstreamObject.SetAnnotations(annotations)
	}
	// Create the resource
	if _, err := c.upstreamClient.Cluster(upstreamLogicalCluster).Resource(gvr).Namespace(upstreamNS).Create(ctx, downstreamObject, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
