/*
Copyright 2022 The KCP Authors.

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

package upsync

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	kcpdynamic "github.com/kcp-dev/client-go/dynamic"
	kcpdynamicinformer "github.com/kcp-dev/client-go/dynamic/dynamicinformer"
	"github.com/kcp-dev/kcp/pkg/logging"
	"github.com/kcp-dev/kcp/pkg/syncer/resourcesync"
	"github.com/kcp-dev/kcp/pkg/syncer/shared"
	"github.com/kcp-dev/kcp/third_party/keyfunctions"
	"github.com/kcp-dev/logicalcluster/v2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	controllerName = "kcp-resource-upsycner"
	labelKey       = "kcp.resource.upsync"
	syncValue      = "sync"
)

type Controller struct {
	queue                     workqueue.RateLimitingInterface
	upstreamClient            kcpdynamic.ClusterInterface
	downstreamClient          dynamic.Interface
	downstreamNamespaceLister cache.GenericLister

	upstreamInformer         kcpdynamicinformer.DynamicSharedInformerFactory
	downstreamUpsyncInformer resourcesync.SyncerInformerFactory
	syncTargetName           string
	syncTargetWorkspace      logicalcluster.Name
	syncTargetUID            types.UID
	syncTargetKey            string
}

type queueKey struct {
	gvr        schema.GroupVersionResource
	key        string
	isUpstream bool
}

func (c *Controller) AddToQueue(gvr schema.GroupVersionResource, obj interface{}, logger logr.Logger, isUpstream bool) {
	key, err := keyfunctions.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	logging.WithQueueKey(logger, key).V(2).Info("queueing GVR", "gvr", gvr.String())
	c.queue.Add(
		queueKey{
			gvr:        gvr,
			key:        key,
			isUpstream: isUpstream,
		},
	)
}

func NewUpSyncer(syncerLogger logr.Logger, syncTargetWorkspace logicalcluster.Name,
	syncTargetName, syncTargetKey string, upstreamClient kcpdynamic.ClusterInterface,
	downstreamClient dynamic.Interface, upstreamInformer kcpdynamicinformer.DynamicSharedInformerFactory,
	downstreamInformers dynamicinformer.DynamicSharedInformerFactory,
	downstreamUpsyncInformer resourcesync.SyncerInformerFactory, syncTargetUID types.UID) (*Controller, error) {
	c := &Controller{
		queue:                     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerName),
		upstreamClient:            upstreamClient,
		downstreamClient:          downstreamClient,
		downstreamNamespaceLister: downstreamInformers.ForResource(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}).Lister(),
		downstreamUpsyncInformer:  downstreamUpsyncInformer,
		syncTargetName:            syncTargetKey,
		syncTargetWorkspace:       syncTargetWorkspace,
		syncTargetUID:             syncTargetUID,
		syncTargetKey:             syncTargetKey,
		upstreamInformer:          upstreamInformer,
	}
	logger := logging.WithReconciler(syncerLogger, controllerName)

	downstreamUpsyncInformer.AddDownstreamEventHandler(
		func(gvr schema.GroupVersionResource) cache.ResourceEventHandler {
			logger.V(2).Info("Set up downstream resources informer", "gvr", gvr.String())
			return cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					c.AddToQueue(gvr, obj, logger, false)
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					c.AddToQueue(gvr, newObj, logger, false)
				},
				DeleteFunc: func(obj interface{}) {
					c.AddToQueue(gvr, obj, logger, false)
				},
			}
		},
	)

	downstreamUpsyncInformer.AddUpstreamEventHandler(
		func(gvr schema.GroupVersionResource) cache.ResourceEventHandler {
			logger.V(2).Info("Set up upstream resources informer", "gvr", gvr.String())
			return cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					metadata, err := meta.Accessor(obj)
					if err != nil {
						logger.Error(err, "Error")
					}
					resource, isUpstream := c.getResourceAndIsUpstream(metadata, gvr, logger)
					c.AddToQueue(gvr, resource, logger, isUpstream)
				},
			}
		},
	)
	return c, nil
}

// If downstream resource is present it will pass it back and return false (isUpstream)
// else it will pass the upstream object and return true
func (c *Controller) getResourceAndIsUpstream(obj metav1.Object, gvr schema.GroupVersionResource, logger logr.Logger) (resource metav1.Object, isUpstream bool) {
	clusterName := logicalcluster.From(obj)
	upstreamNamespace := obj.GetNamespace()
	desiredNSLocator := shared.NewNamespaceLocator(clusterName, c.syncTargetWorkspace, c.syncTargetUID, c.syncTargetName, obj.GetNamespace())
	downstreamNamespace := ""
	var err error

	object := &metav1.ObjectMeta{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	if upstreamNamespace != "" {
		downstreamNamespace, err = shared.PhysicalClusterNamespaceName(desiredNSLocator)
		if err != nil {
			logger.Error(err, "Error")
			return object, true
		}
		_, err := c.downstreamNamespaceLister.Get(downstreamNamespace)
		if err != nil {
			return object, true
		}
		downstreamObj, err := c.downstreamClient.Resource(gvr).Namespace(downstreamNamespace).Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return object, true
		}
		return &metav1.ObjectMeta{Name: downstreamObj.GetName(), Namespace: downstreamObj.GetNamespace()}, false
	} else {
		downstreamObj, err := c.downstreamClient.Resource(gvr).Get(context.TODO(), obj.GetName(), metav1.GetOptions{})
		if err != nil {
			return object, true
		}
		return &metav1.ObjectMeta{Name: downstreamObj.GetName(), Namespace: downstreamObj.GetNamespace()}, false

	}
}

func (c *Controller) Start(ctx context.Context, numThreads int) {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	logger := logging.WithReconciler(klog.FromContext(ctx), controllerName)
	ctx = klog.NewContext(ctx, logger)
	logger.Info("Starting upsync workers")
	defer logger.Info("Stopping upsync workers")

	for i := 0; i < numThreads; i++ {
		go wait.UntilWithContext(ctx, c.startWorker, time.Second)
	}
	<-ctx.Done()
}

func (c *Controller) startWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	qk := key.(queueKey)

	logger := logging.WithQueueKey(klog.FromContext(ctx), qk.key).WithValues("gvr", qk.gvr.String())
	ctx = klog.NewContext(ctx, logger)
	logger.V(1).Info("Processing key")
	defer c.queue.Done(key)

	if err := c.process(ctx, qk.gvr, qk.key, qk.isUpstream); err != nil {
		runtime.HandleError(fmt.Errorf("%s failed to upsync %q, err: %w", controllerName, key, err))
		c.queue.AddRateLimited(key)
		return true
	}

	c.queue.Forget(key)

	return true
}
