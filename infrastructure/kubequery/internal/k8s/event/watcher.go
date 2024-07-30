/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package event

import (
	"context"
	"sync"
	"time"

	osquery "github.com/Uptycs/basequery-go"
	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const tableName = "kubernetes_events"

// Watcher holds the kubernetes informer. Can be started to receive events from k8s.
type Watcher struct {
	lock     sync.Mutex
	client   *osquery.ExtensionManagerClient
	stopper  chan struct{}
	informer cache.SharedInformer
}

type event struct {
	Time                metav1.Time
	EventType           string
	ClusterUID          types.UID
	ClusterName         string
	Name                string
	Namespace           string
	CreationTimestamp   metav1.Time
	Labels              map[string]string
	Annotations         map[string]string
	ReportingController string
	ReportingInstance   string
	Action              string
	Reason              string
	Note                string
	Type                string
	RegardingKind       string
	RegardingNamespace  string
	RegardingName       string
	RegardingUID        types.UID
	RelatedKind         string
	RelatedNamespace    string
	RelatedName         string
	RelatedUID          types.UID
}

// Columns returns kubernetes event fields as Osquery table columns.
func Columns() []table.ColumnDefinition {
	return k8s.GetSchema(&event{})
}

// Generate generates the kubernetes events as Osquery table data.
// For event'ed table Generate method should never be called. So this always returns nil.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return nil, nil
}

func streamEvent(client *osquery.ExtensionManagerClient, eventType string, e *v1.Event) {
	event := &event{
		EventType:           eventType,
		ClusterName:         k8s.GetClusterName(),
		ClusterUID:          k8s.GetClusterUID(),
		Name:                e.Name,
		Namespace:           e.Namespace,
		CreationTimestamp:   e.CreationTimestamp,
		Reason:              e.Reason,
		Note:                e.Note,
		Type:                e.Type,
		Labels:              e.Labels,
		Annotations:         e.Annotations,
		ReportingController: e.ReportingController,
		ReportingInstance:   e.ReportingInstance,
		Action:              e.Action,
		RegardingUID:        e.Regarding.UID,
		RegardingKind:       e.Regarding.Kind,
		RegardingName:       e.Regarding.Name,
		RegardingNamespace:  e.Regarding.Namespace,
	}
	if e.EventTime.IsZero() {
		event.Time = metav1.Now()
	} else {
		event.Time = metav1.Time(e.EventTime)
	}
	if e.Related != nil {
		event.RelatedUID = e.Related.UID
		event.RelatedKind = e.Related.Kind
		event.RelatedName = e.Related.Name
		event.RelatedNamespace = e.Related.Namespace
	}

	events := make([]map[string]string, 1)
	events[0] = k8s.ToMap(event)

	// TODO: Returned status, error is ignored
	client.StreamEvents(tableName, events)
}

// CreateEventWatcher when started will get events from kubernetes that will be streamed to Osquery.
func CreateEventWatcher(socket string, timeout time.Duration) (*Watcher, error) {
	client, err := osquery.NewClient(socket, timeout)
	if err != nil {
		return nil, err
	}

	factory := informers.NewSharedInformerFactory(k8s.GetClient(), 0)
	watcher := &Watcher{
		client:   client,
		stopper:  make(chan struct{}),
		informer: factory.Events().V1().Events().Informer(),
	}

	watcher.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			watcher.lock.Lock()
			streamEvent(client, "add", obj.(*v1.Event))
			watcher.lock.Unlock()
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			watcher.lock.Lock()
			streamEvent(client, "update", new.(*v1.Event))
			watcher.lock.Unlock()
		},
		DeleteFunc: func(obj interface{}) {
			watcher.lock.Lock()
			streamEvent(client, "delete", obj.(*v1.Event))
			watcher.lock.Unlock()
		},
	})

	return watcher, nil
}

// Start will start the watcher to stream kubernetes events as they come in.
func (e *Watcher) Start() {
	go e.informer.Run(e.stopper)
}

// Stop terminates the watcher.
func (e *Watcher) Stop() {
	e.client.Close()
	e.stopper <- struct{}{}
	close(e.stopper)
}
