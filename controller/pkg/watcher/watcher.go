package watcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// WatchedResource defines a GVR to watch with a human-friendly label.
type WatchedResource struct {
	GVR   schema.GroupVersionResource
	Label string
}

// DefaultWatchedResources returns the list of all resources the governance
// controller cares about. A change to any of these triggers a reconcile.
func DefaultWatchedResources() []WatchedResource {
	return []WatchedResource{
		{
			Label: "Gateway",
			GVR: schema.GroupVersionResource{
				Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways",
			},
		},
		{
			Label: "HTTPRoute",
			GVR: schema.GroupVersionResource{
				Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes",
			},
		},
		{
			Label: "AgentgatewayBackend",
			GVR: schema.GroupVersionResource{
				Group: "agentgateway.dev", Version: "v1alpha1", Resource: "agentgatewaybackends",
			},
		},
		{
			Label: "AgentgatewayPolicy",
			GVR: schema.GroupVersionResource{
				Group: "agentgateway.dev", Version: "v1alpha1", Resource: "agentgatewaypolicies",
			},
		},
		{
			Label: "MCPServer",
			GVR: schema.GroupVersionResource{
				Group: "kagent.dev", Version: "v1alpha2", Resource: "mcpservers",
			},
		},
		{
			Label: "RemoteMCPServer",
			GVR: schema.GroupVersionResource{
				Group: "kagent.dev", Version: "v1alpha2", Resource: "remotemcpservers",
			},
		},
		{
			Label: "Agent",
			GVR: schema.GroupVersionResource{
				Group: "kagent.dev", Version: "v1alpha2", Resource: "agents",
			},
		},
		{
			Label: "MCPGovernancePolicy",
			GVR: schema.GroupVersionResource{
				Group: "governance.mcp.io", Version: "v1alpha1", Resource: "mcpgovernancepolicies",
			},
		},
		{
			Label: "MCPServerCatalog",
			GVR: schema.GroupVersionResource{
				Group: "agentregistry.dev", Version: "v1alpha1", Resource: "mcpservercatalogs",
			},
		},
	}
}

// ReconcileFunc is the callback invoked when any watched resource changes.
// The reason string describes what triggered the reconcile.
type ReconcileFunc func(reason string)

// ResourceWatcher watches Kubernetes resources and triggers reconciliation
// with debouncing when changes are detected.
type ResourceWatcher struct {
	dynClient  dynamic.Interface
	reconcile  ReconcileFunc
	debounce   time.Duration
	resyncPeriod time.Duration

	mu            sync.Mutex
	debounceTimer *time.Timer
	pendingReason string
	watchedGVRs   []WatchedResource
	activeCount   int // how many GVRs are being actively watched
	stopCh        chan struct{}
	stopped       bool

	// Stats
	statsMu        sync.RWMutex
	lastEvent      time.Time
	lastReconcile  time.Time
	eventCount     int64
	reconcileCount int64
	watchErrors    []string
}

// Config holds the configuration for the ResourceWatcher.
type Config struct {
	// DynamicClient is the Kubernetes dynamic client.
	DynamicClient dynamic.Interface

	// Reconcile is called when any watched resource changes.
	Reconcile ReconcileFunc

	// Debounce is the delay to wait after the last change event before
	// triggering a reconcile. Multiple rapid changes are batched.
	// Default: 3 seconds.
	Debounce time.Duration

	// ResyncPeriod is how often informers do a full re-list from the API
	// server, even if no watch events have been received. This acts as a
	// safety net in case a watch event is missed.
	// Default: 5 minutes.
	ResyncPeriod time.Duration

	// WatchedResources is the list of GVRs to watch. If nil, uses
	// DefaultWatchedResources().
	WatchedResources []WatchedResource
}

// New creates and returns a new ResourceWatcher. Call Start() to begin watching.
func New(cfg Config) (*ResourceWatcher, error) {
	if cfg.DynamicClient == nil {
		return nil, fmt.Errorf("DynamicClient is required")
	}
	if cfg.Reconcile == nil {
		return nil, fmt.Errorf("Reconcile callback is required")
	}
	if cfg.Debounce <= 0 {
		cfg.Debounce = 3 * time.Second
	}
	if cfg.ResyncPeriod <= 0 {
		cfg.ResyncPeriod = 5 * time.Minute
	}
	if cfg.WatchedResources == nil {
		cfg.WatchedResources = DefaultWatchedResources()
	}

	return &ResourceWatcher{
		dynClient:    cfg.DynamicClient,
		reconcile:    cfg.Reconcile,
		debounce:     cfg.Debounce,
		resyncPeriod: cfg.ResyncPeriod,
		watchedGVRs:  cfg.WatchedResources,
		stopCh:       make(chan struct{}),
	}, nil
}

// Start begins watching all configured resources. It blocks until ctx is
// cancelled or Stop() is called. Typically called in a goroutine.
func (w *ResourceWatcher) Start(ctx context.Context) {
	log.Printf("[watcher] Starting resource watcher with %d resource types, debounce=%v, resync=%v",
		len(w.watchedGVRs), w.debounce, w.resyncPeriod)

	factory := dynamicinformer.NewDynamicSharedInformerFactory(w.dynClient, w.resyncPeriod)

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { w.onEvent("add", obj) },
		UpdateFunc: func(old, new interface{}) {
			// Skip status-only updates by comparing metadata.generation.
			// When only .status changes, the generation stays the same.
			// This prevents a reconcile loop: scan → update status → watch event → scan → ...
			oldU, oldOk := old.(*unstructured.Unstructured)
			newU, newOk := new.(*unstructured.Unstructured)
			if oldOk && newOk && oldU.GetGeneration() == newU.GetGeneration() && oldU.GetGeneration() > 0 {
				return // status-only update, skip
			}
			w.onEvent("update", new)
		},
		DeleteFunc: func(obj interface{}) { w.onEvent("delete", obj) },
	}

	for _, wr := range w.watchedGVRs {
		informer := factory.ForResource(wr.GVR).Informer()
		_, err := informer.AddEventHandler(handler)
		if err != nil {
			log.Printf("[watcher] WARNING: Failed to add handler for %s: %v", wr.Label, err)
			w.recordWatchError(fmt.Sprintf("Failed to watch %s: %v", wr.Label, err))
			continue
		}
		w.mu.Lock()
		w.activeCount++
		w.mu.Unlock()
		log.Printf("[watcher] Watching %s (%s)", wr.Label, wr.GVR.Resource)
	}

	// Start all informers
	factory.Start(w.stopCh)

	// Wait for initial cache sync
	log.Printf("[watcher] Waiting for informer caches to sync...")
	factory.WaitForCacheSync(w.stopCh)
	log.Printf("[watcher] All informer caches synced — watching for changes")

	// Block until stopped
	select {
	case <-ctx.Done():
		w.Stop()
	case <-w.stopCh:
	}
}

// Stop signals the watcher to shut down.
func (w *ResourceWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.stopCh)
		if w.debounceTimer != nil {
			w.debounceTimer.Stop()
		}
		log.Printf("[watcher] Resource watcher stopped")
	}
}

// Stats returns the current watcher statistics.
func (w *ResourceWatcher) Stats() WatcherStats {
	w.statsMu.RLock()
	defer w.statsMu.RUnlock()
	w.mu.Lock()
	active := w.activeCount
	w.mu.Unlock()
	return WatcherStats{
		ActiveWatches:  active,
		TotalGVRs:      len(w.watchedGVRs),
		LastEvent:      w.lastEvent,
		LastReconcile:  w.lastReconcile,
		EventCount:     w.eventCount,
		ReconcileCount: w.reconcileCount,
		WatchErrors:    append([]string{}, w.watchErrors...),
	}
}

// WatcherStats holds runtime statistics about the watcher.
type WatcherStats struct {
	ActiveWatches  int       `json:"activeWatches"`
	TotalGVRs      int       `json:"totalGVRs"`
	LastEvent      time.Time `json:"lastEvent"`
	LastReconcile  time.Time `json:"lastReconcile"`
	EventCount     int64     `json:"eventCount"`
	ReconcileCount int64     `json:"reconcileCount"`
	WatchErrors    []string  `json:"watchErrors,omitempty"`
}

// onEvent is called by the informer for every add/update/delete event.
// It debounces rapid changes so the reconcile function is called only once
// after a burst of events settles.
func (w *ResourceWatcher) onEvent(eventType string, obj interface{}) {
	// Extract resource info for logging
	reason := eventType
	if u, ok := obj.(interface {
		GetName() string
		GetNamespace() string
	}); ok {
		reason = fmt.Sprintf("%s %s/%s", eventType, u.GetNamespace(), u.GetName())
	}

	w.statsMu.Lock()
	w.lastEvent = time.Now()
	w.eventCount++
	w.statsMu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}

	// Accumulate reasons
	if w.pendingReason == "" {
		w.pendingReason = reason
	} else {
		w.pendingReason = fmt.Sprintf("%s; %s", w.pendingReason, reason)
	}

	// Reset the debounce timer
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceTimer = time.AfterFunc(w.debounce, func() {
		w.mu.Lock()
		reason := w.pendingReason
		w.pendingReason = ""
		w.mu.Unlock()

		w.statsMu.Lock()
		w.lastReconcile = time.Now()
		w.reconcileCount++
		w.statsMu.Unlock()

		log.Printf("[watcher] Reconciling — triggered by: %s", reason)
		w.reconcile(reason)
	})
}

func (w *ResourceWatcher) recordWatchError(err string) {
	w.statsMu.Lock()
	defer w.statsMu.Unlock()
	w.watchErrors = append(w.watchErrors, err)
	// Keep only last 10 errors
	if len(w.watchErrors) > 10 {
		w.watchErrors = w.watchErrors[len(w.watchErrors)-10:]
	}
}
