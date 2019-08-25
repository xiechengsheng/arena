// Copyright 2018 The Kubeflow Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by informer-gen. DO NOT EDIT.

// This file was automatically generated by informer-gen

package v1alpha1

import (
	time "time"

	kubeflowv1alpha1 "github.com/kubeflow/arena/pkg/operators/mpi-operator/apis/kubeflow/v1alpha1"
	versioned "github.com/kubeflow/arena/pkg/operators/mpi-operator/client/clientset/versioned"
	internalinterfaces "github.com/kubeflow/arena/pkg/operators/mpi-operator/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kubeflow/arena/pkg/operators/mpi-operator/client/listers/kubeflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// MPIJobInformer provides access to a shared informer and lister for
// MPIJobs.
type MPIJobInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.MPIJobLister
}

type mPIJobInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewMPIJobInformer constructs a new informer for MPIJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMPIJobInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.KubeflowV1alpha1().MPIJobs(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.KubeflowV1alpha1().MPIJobs(namespace).Watch(options)
			},
		},
		&kubeflowv1alpha1.MPIJob{},
		resyncPeriod,
		indexers,
	)
}

func defaultMPIJobInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewMPIJobInformer(client, v1.NamespaceAll, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *mPIJobInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kubeflowv1alpha1.MPIJob{}, defaultMPIJobInformer)
}

func (f *mPIJobInformer) Lister() v1alpha1.MPIJobLister {
	return v1alpha1.NewMPIJobLister(f.Informer().GetIndexer())
}