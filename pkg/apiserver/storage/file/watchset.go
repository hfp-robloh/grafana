// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/tilt-dev/tilt-apiserver/blob/main/pkg/storage/filepath/watchset.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package file

import (
	"fmt"
	"strconv"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"
)

const (
	UpdateChannelSize         = 10
	InitialWatchNodesSize     = 20
	InitialBufferedEventsSize = 25
)

type eventWrapper struct {
	ev watch.Event
	// optional: oldObject is only set for modifications for determining their type as necessary (when using predicate filtering)
	oldObject runtime.Object
}

type watchNode struct {
	s           *WatchSet
	id          int
	updateCh    chan eventWrapper
	outCh       chan watch.Event
	requestedRV uint64
	// the watch may or may not be namespaced for a namespaced resource. This is always nil for cluster-scoped kinds
	watchNamespace *string
	predicate      storage.SelectionPredicate
	// Buffers events during startup so that the brief window in which the async
	// part of start method starts doesn't lead to us missing events
	buffered      []eventWrapper
	bufferedMutex sync.RWMutex
	started       bool
}

// Keeps track of which watches need to be notified
type WatchSet struct {
	mu sync.RWMutex
	// mu protects both nodes and counter
	nodes   map[int]*watchNode
	counter int
}

func NewWatchSet() *WatchSet {
	return &WatchSet{
		nodes:   make(map[int]*watchNode, InitialWatchNodesSize),
		counter: 0,
	}
}

// Creates a new watch with a unique id, but
// does not start sending events to it until start() is called.
func (s *WatchSet) newWatch(requestedRV uint64, p storage.SelectionPredicate, namespace *string) *watchNode {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++

	node := &watchNode{
		requestedRV: requestedRV,
		id:          s.counter,
		s:           s,
		buffered:    make([]eventWrapper, 0, InitialBufferedEventsSize),
		// updateCh size needs to be > 1 to allow slower clients to not block passing new events
		updateCh: make(chan eventWrapper, UpdateChannelSize),
		// outCh size needs to be > 1 for single process use-cases such as tests where watch and event seeding from CUD
		// events is happening on the same thread
		outCh:          make(chan watch.Event, UpdateChannelSize),
		predicate:      p,
		watchNamespace: namespace,
	}

	return node
}

func (s *WatchSet) cleanupWatchers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.nodes {
		w.stop()

	}
}

// oldObject is only passed in the event of a modification
// in case a predicate filtered watch is impacted as a result of modification
func (s *WatchSet) notifyWatchers(ev watch.Event, oldObject runtime.Object) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	updateEv := eventWrapper{
		ev: ev,
	}
	if oldObject != nil {
		updateEv.oldObject = oldObject
	}

	for _, w := range s.nodes {
		// Events are buffered per node before startup is complete
		// this is because of an inadvertent delay which is built into the watch process
		// While a watch begins and gets subscribed fully, another client (internal or external) could
		// change system state and this may be after the informers have successfully listed state
		if !w.started {
			w.bufferedMutex.Lock()
			w.buffered = append(w.buffered, updateEv)
			w.bufferedMutex.Unlock()
		}

		w.started = true
		w.updateCh <- updateEv
	}
}

// isValid is not necessary to be called on oldObject in UpdateEvents - assuming the Watch pushes correctly setup eventWrapper our way
// first bool is whether the event is valid for current watcher
// second bool is whether checking the old value against the predicate may be valuable to the caller
// second bool may be a helpful aid to establish context around MODIFIED events
// (note that this second bool is only marked true if we pass other checks first, namely RV and namespace)
func (w *watchNode) isValid(e eventWrapper) (bool, bool, error) {
	obj, err := meta.Accessor(e.ev.Object)
	if err != nil {
		klog.Warningf("Could not get accessor to object in event")
		return false, false, nil
	}

	eventRV, err := strconv.Atoi(obj.GetResourceVersion())
	if err != nil {
		return false, false, err
	}

	if eventRV <= int(w.requestedRV) {
		return false, false, nil
	}

	if w.watchNamespace != nil && *w.watchNamespace != obj.GetNamespace() {
		return false, false, err
	}

	valid, err := w.predicate.Matches(e.ev.Object)
	if err != nil {
		return false, false, err
	}

	return valid, true, nil
}

// Only call this method if current object matches the predicate
func (w *watchNode) handleAddedForFilteredList(e eventWrapper) (*watch.Event, error) {
	if e.oldObject == nil {
		return nil, fmt.Errorf("oldObject should be set for modified events")
	}

	ok, err := w.predicate.Matches(e.oldObject)
	if err != nil {
		return nil, err
	}

	if !ok {
		e.ev.Type = watch.Added
		return &e.ev, nil
	}

	return nil, nil
}

func (w *watchNode) handleDeletedForFilteredList(e eventWrapper) (*watch.Event, error) {
	if e.oldObject == nil {
		return nil, fmt.Errorf("oldObject should be set for modified events")
	}

	ok, err := w.predicate.Matches(e.oldObject)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	// isn't a match but used to be
	e.ev.Type = watch.Deleted

	oldObjectAccessor, err := meta.Accessor(e.oldObject)
	if err != nil {
		klog.Warning("Could not get accessor to correct the old RV of filtered out object")
		return nil, err
	}

	obj, err := meta.Accessor(e.ev.Object)
	if err != nil {
		klog.Warningf("Could not get accessor to object in event")
		return nil, err
	}

	oldObjectAccessor.SetResourceVersion(obj.GetResourceVersion())
	e.ev.Object = e.oldObject

	return &e.ev, nil
}

func (w *watchNode) processEvent(e eventWrapper) error {
	valid, runDeleteFromFilteredListHandler, err := w.isValid(e)
	if err != nil {
		klog.Errorf("Could not determine validity of the event: %v", err)
		return err
	}
	if valid {
		// klog.Infof("Event %v is valid", e.ev)
		if e.ev.Type == watch.Modified {
			ev, err := w.handleAddedForFilteredList(e)
			if err != nil {
				return err
			}
			if ev != nil {
				w.outCh <- *ev
			} else {
				// forward the original event if add handling didn't signal any impact
				w.outCh <- e.ev
			}
		} else {
			w.outCh <- e.ev
		}
	} else if runDeleteFromFilteredListHandler {
		if e.ev.Type == watch.Modified {
			ev, err := w.handleDeletedForFilteredList(e)
			// klog.Infof("Result of handleDeletedForFilteredList: ev=%v, err=%v", ev, err)
			if err != nil {
				return err
			}
			if ev != nil {
				w.outCh <- *ev
			}
		} // explicitly doesn't have an event forward for the else case here
	}

	return nil
}

// Start sending events to this watch.
func (w *watchNode) Start(initEvents ...watch.Event) {
	w.s.mu.Lock()
	w.s.nodes[w.id] = w
	w.s.mu.Unlock()

	go func() {
		for _, ev := range initEvents {
			if err := w.processEvent(eventWrapper{ev: ev}); err != nil {
				klog.Errorf("Could not process event: %v", err)
			}
		}

		// The if check below helps not send duplicate events when reading from 0
		// since ADDED events made from initial list above are already sent
		if w.requestedRV != 0 {
			w.bufferedMutex.RLock()
			for _, e := range w.buffered {
				if err := w.processEvent(e); err != nil {
					klog.Errorf("Could not process event: %v", err)
				}
			}
			w.bufferedMutex.RUnlock()
		}

		for e := range w.updateCh {
			if err := w.processEvent(e); err != nil {
				klog.Errorf("Could not process event: %v", err)
			}
		}
		close(w.outCh)
	}()
}

func (w *watchNode) Stop() {
	w.s.mu.Lock()
	defer w.s.mu.Unlock()
	w.stop()
}

// Unprotected func: ensure mutex on the parent watch set is locked before calling
func (w *watchNode) stop() {
	delete(w.s.nodes, w.id)
	close(w.updateCh)
}

func (w *watchNode) ResultChan() <-chan watch.Event {
	return w.outCh
}
