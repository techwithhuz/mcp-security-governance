package inventory

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// MCPServerCatalogGVR is the GroupVersionResource for the agentregistry MCPServerCatalog.
var MCPServerCatalogGVR = schema.GroupVersionResource{
	Group:    "agentregistry.dev",
	Version:  "v1alpha1",
	Resource: "mcpservercatalogs",
}

// Watcher watches MCPServerCatalog resources from the Agent Registry inventory
// and scores each one with a Verified Score upon add/update/delete.
type Watcher struct {
	dynClient    dynamic.Interface
	policy       ScoringPolicy
	namespace    string // namespace to watch (empty = all namespaces)
	patcher      *StatusPatcher // patches status.publisher field

	mu        sync.RWMutex
	resources map[string]*VerifiedResource // key: "namespace/name"
	summary   VerifiedSummary

	// Stats
	statsMu       sync.RWMutex
	lastEvent     time.Time
	lastReconcile time.Time
	eventCount    int64
	reconcileCount int64

	// Lifecycle
	stopCh  chan struct{}
	stopped bool

	// OnChange is called whenever verified resources are updated.
	// Can be used to trigger downstream actions (e.g. re-score cluster).
	OnChange func()
	
	// PatchStatusOnUpdate controls whether to patch resource status after scoring
	PatchStatusOnUpdate bool
}

// WatcherConfig configures the inventory watcher.
type WatcherConfig struct {
	// DynamicClient is the Kubernetes dynamic client.
	DynamicClient dynamic.Interface
	// ScoringPolicy configures scoring thresholds.
	Policy ScoringPolicy
	// Namespace to watch. Empty string means all namespaces.
	Namespace string
	// OnChange callback when resources are reconciled.
	OnChange func()
	// PatchStatusOnUpdate controls whether to patch resource status after scoring.
	// If true, the watcher will update the .status.publisher field with governance scores.
	PatchStatusOnUpdate bool
}

// NewWatcher creates a new inventory watcher.
func NewWatcher(cfg WatcherConfig) (*Watcher, error) {
	if cfg.DynamicClient == nil {
		return nil, fmt.Errorf("DynamicClient is required")
	}
	return &Watcher{
		dynClient: cfg.DynamicClient,
		patcher:   NewStatusPatcher(cfg.DynamicClient),
		policy:    cfg.Policy,
		namespace: cfg.Namespace,
		resources: make(map[string]*VerifiedResource),
		stopCh:    make(chan struct{}),
		OnChange:  cfg.OnChange,
		PatchStatusOnUpdate: cfg.PatchStatusOnUpdate,
	}, nil
}

// Start begins watching MCPServerCatalog resources. Blocks until ctx is
// cancelled or Stop() is called.
func (w *Watcher) Start(ctx context.Context) {
	log.Printf("[inventory] Starting MCPServerCatalog watcher (namespace=%q)", w.namespace)

	var factory dynamicinformer.DynamicSharedInformerFactory
	if w.namespace != "" {
		factory = dynamicinformer.NewFilteredDynamicSharedInformerFactory(
			w.dynClient, 5*time.Minute, w.namespace, nil,
		)
	} else {
		factory = dynamicinformer.NewDynamicSharedInformerFactory(w.dynClient, 5*time.Minute)
	}

	informer := factory.ForResource(MCPServerCatalogGVR).Informer()

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			w.onAdd(u)
		},
		UpdateFunc: func(old, new interface{}) {
			oldU, oldOk := old.(*unstructured.Unstructured)
			newU, newOk := new.(*unstructured.Unstructured)
			if !oldOk || !newOk {
				return
			}
			// Skip status-only updates (generation unchanged)
			if oldU.GetGeneration() == newU.GetGeneration() && oldU.GetGeneration() > 0 {
				// But still re-score if resourceVersion changed (status update may
				// include deployment health changes we want to pick up)
				if oldU.GetResourceVersion() != newU.GetResourceVersion() {
					w.onUpdate(newU)
				}
				return
			}
			w.onUpdate(newU)
		},
		DeleteFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				// Handle tombstone
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					return
				}
				u, ok = tombstone.Obj.(*unstructured.Unstructured)
				if !ok {
					return
				}
			}
			w.onDelete(u)
		},
	}

	_, err := informer.AddEventHandler(handler)
	if err != nil {
		log.Printf("[inventory] WARNING: Failed to add event handler: %v", err)
		return
	}

	// Start the informer
	factory.Start(w.stopCh)

	// Wait for cache sync
	log.Printf("[inventory] Waiting for MCPServerCatalog informer cache sync...")
	factory.WaitForCacheSync(w.stopCh)
	log.Printf("[inventory] MCPServerCatalog cache synced — watching for changes")

	// Block until stopped
	select {
	case <-ctx.Done():
		w.Stop()
	case <-w.stopCh:
	}
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.stopCh)
		log.Printf("[inventory] MCPServerCatalog watcher stopped")
	}
}

// GetResources returns a snapshot of all verified resources.
func (w *Watcher) GetResources() []VerifiedResource {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make([]VerifiedResource, 0, len(w.resources))
	for _, r := range w.resources {
		result = append(result, *r)
	}
	return result
}

// GetResource returns a single verified resource by namespace/name.
func (w *Watcher) GetResource(namespace, name string) (*VerifiedResource, bool) {
	key := namespace + "/" + name
	w.mu.RLock()
	defer w.mu.RUnlock()
	r, ok := w.resources[key]
	if !ok {
		return nil, false
	}
	copy := *r
	return &copy, true
}

// GetSummary returns the current verified summary.
func (w *Watcher) GetSummary() VerifiedSummary {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.summary
}

// Stats returns watcher runtime stats.
func (w *Watcher) Stats() WatcherStats {
	w.statsMu.RLock()
	defer w.statsMu.RUnlock()
	return WatcherStats{
		LastEvent:      w.lastEvent,
		LastReconcile:  w.lastReconcile,
		EventCount:     w.eventCount,
		ReconcileCount: w.reconcileCount,
		ResourceCount:  len(w.resources),
	}
}

// WatcherStats holds runtime statistics.
type WatcherStats struct {
	LastEvent      time.Time `json:"lastEvent"`
	LastReconcile  time.Time `json:"lastReconcile"`
	EventCount     int64     `json:"eventCount"`
	ReconcileCount int64     `json:"reconcileCount"`
	ResourceCount  int       `json:"resourceCount"`
}

// ---------- Event Handlers ----------

func (w *Watcher) onAdd(obj *unstructured.Unstructured) {
	w.statsMu.Lock()
	w.lastEvent = time.Now()
	w.eventCount++
	w.statsMu.Unlock()

	res := w.extractResource(obj)
	res.VerifiedScore = ScoreCatalog(res, w.policy)
	res.LastScored = time.Now()

	key := res.Namespace + "/" + res.Name
	w.mu.Lock()
	w.resources[key] = res
	w.mu.Unlock()

	w.recomputeSummary()

	log.Printf("[inventory] MCPServerCatalog ADDED: %s — Verified Score: %d (%s) Grade: %s",
		key, res.VerifiedScore.Score, res.VerifiedScore.Status, res.VerifiedScore.Grade)

	// Patch the resource status if enabled
	if w.PatchStatusOnUpdate {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := w.patcher.PatchCatalogStatus(ctx, res); err != nil {
			log.Printf("[inventory] WARNING: Failed to patch status for %s: %v", key, err)
		}
	}

	w.statsMu.Lock()
	w.lastReconcile = time.Now()
	w.reconcileCount++
	w.statsMu.Unlock()

	if w.OnChange != nil {
		w.OnChange()
	}
}

func (w *Watcher) onUpdate(obj *unstructured.Unstructured) {
	w.statsMu.Lock()
	w.lastEvent = time.Now()
	w.eventCount++
	w.statsMu.Unlock()

	res := w.extractResource(obj)

	key := res.Namespace + "/" + res.Name
	w.mu.RLock()
	existing, exists := w.resources[key]
	w.mu.RUnlock()

	// Skip if resource version hasn't changed
	if exists && existing.ResourceVersion == res.ResourceVersion {
		return
	}

	res.VerifiedScore = ScoreCatalog(res, w.policy)
	res.LastScored = time.Now()

	w.mu.Lock()
	w.resources[key] = res
	w.mu.Unlock()

	w.recomputeSummary()

	log.Printf("[inventory] MCPServerCatalog UPDATED: %s — Verified Score: %d (%s) Grade: %s",
		key, res.VerifiedScore.Score, res.VerifiedScore.Status, res.VerifiedScore.Grade)

	// Patch the resource status if enabled
	if w.PatchStatusOnUpdate {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := w.patcher.PatchCatalogStatus(ctx, res); err != nil {
			log.Printf("[inventory] WARNING: Failed to patch status for %s: %v", key, err)
		}
	}

	w.statsMu.Lock()
	w.lastReconcile = time.Now()
	w.reconcileCount++
	w.statsMu.Unlock()

	if w.OnChange != nil {
		w.OnChange()
	}
}

func (w *Watcher) onDelete(obj *unstructured.Unstructured) {
	w.statsMu.Lock()
	w.lastEvent = time.Now()
	w.eventCount++
	w.statsMu.Unlock()

	key := obj.GetNamespace() + "/" + obj.GetName()
	w.mu.Lock()
	delete(w.resources, key)
	w.mu.Unlock()

	w.recomputeSummary()

	log.Printf("[inventory] MCPServerCatalog DELETED: %s — removed from verified resources", key)

	w.statsMu.Lock()
	w.lastReconcile = time.Now()
	w.reconcileCount++
	w.statsMu.Unlock()

	if w.OnChange != nil {
		w.OnChange()
	}
}

// ---------- Resource Extraction ----------

func (w *Watcher) extractResource(obj *unstructured.Unstructured) *VerifiedResource {
	labels := obj.GetLabels()
	res := &VerifiedResource{
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		ResourceVersion: obj.GetResourceVersion(),
		SourceKind:      labels["agentregistry.dev/source-kind"],
		SourceName:      labels["agentregistry.dev/source-name"],
		SourceNamespace: labels["agentregistry.dev/source-namespace"],
		Environment:     labels["agentregistry.dev/environment"],
		Cluster:         labels["agentregistry.dev/cluster"],
	}

	// Extract spec fields
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec != nil {
		res.CatalogName, _, _ = unstructured.NestedString(obj.Object, "spec", "name")
		res.Title, _, _ = unstructured.NestedString(obj.Object, "spec", "title")
		res.Description, _, _ = unstructured.NestedString(obj.Object, "spec", "description")
		res.Version, _, _ = unstructured.NestedString(obj.Object, "spec", "version")

		// Extract packages info
		packages, _, _ := unstructured.NestedSlice(obj.Object, "spec", "packages")
		for _, p := range packages {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if img, ok := pm["identifier"].(string); ok && res.PackageImage == "" {
				res.PackageImage = img
			}
			transport, _ := pm["transport"].(map[string]interface{})
			if transport != nil {
				if t, ok := transport["type"].(string); ok && res.Transport == "" {
					res.Transport = t
				}
			}
		}

		// Extract remotes info
		remotes, _, _ := unstructured.NestedSlice(obj.Object, "spec", "remotes")
		for _, r := range remotes {
			rm, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			if url, ok := rm["url"].(string); ok && res.RemoteURL == "" {
				res.RemoteURL = url
			}
			if t, ok := rm["type"].(string); ok && res.Transport == "" {
				res.Transport = t
			}
		}
	}

	// Extract status fields
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status != nil {
		res.Published, _, _ = unstructured.NestedBool(obj.Object, "status", "published")
		res.ManagementType, _, _ = unstructured.NestedString(obj.Object, "status", "managementType")

		// Deployment readiness
		deployReady, _, _ := unstructured.NestedBool(obj.Object, "status", "deployment", "ready")
		res.DeploymentReady = deployReady

		// Used-by agents
		usedBySlice, _, _ := unstructured.NestedSlice(obj.Object, "status", "usedBy")
		for _, u := range usedBySlice {
			um, ok := u.(map[string]interface{})
			if !ok {
				continue
			}
			usage := AgentUsage{}
			if n, ok := um["name"].(string); ok {
				usage.Name = n
			}
			if ns, ok := um["namespace"].(string); ok {
				usage.Namespace = ns
			}
			if tools, ok := um["toolNames"].([]interface{}); ok {
				for _, t := range tools {
					if ts, ok := t.(string); ok {
						usage.ToolNames = append(usage.ToolNames, ts)
					}
				}
				res.ToolCount += len(usage.ToolNames)
			}
			res.UsedByAgents = append(res.UsedByAgents, usage)
		}

		// Collect tool names from usedBy
		toolSet := make(map[string]bool)
		for _, u := range res.UsedByAgents {
			for _, t := range u.ToolNames {
				toolSet[t] = true
			}
		}
		for t := range toolSet {
			res.ToolNames = append(res.ToolNames, t)
		}
		if res.ToolCount == 0 {
			res.ToolCount = len(res.ToolNames)
		}
	}

	return res
}

// ---------- Summary ----------

func (w *Watcher) recomputeSummary() {
	w.mu.Lock()
	defer w.mu.Unlock()

	s := VerifiedSummary{
		LastReconcile: time.Now(),
	}

	totalScore := 0
	verifiedThreshold := policyVerifiedThreshold(w.policy)
	unverifiedThreshold := policyUnverifiedThreshold(w.policy)
	for _, r := range w.resources {
		s.TotalCatalogs++
		s.TotalScored++
		totalScore += r.VerifiedScore.Score
		s.TotalTools += r.ToolCount
		s.TotalAgentUsages += len(r.UsedByAgents)

		switch {
		case r.VerifiedScore.Score >= verifiedThreshold:
			s.VerifiedCount++
		case r.VerifiedScore.Score >= unverifiedThreshold:
			s.UnverifiedCount++
			s.WarningCount++
		default:
			s.RejectedCount++
			s.CriticalCount++
		}
	}

	if s.TotalCatalogs > 0 {
		s.AverageScore = totalScore / s.TotalCatalogs
	}

	w.summary = s
}

// ---------- Utility ----------

// ParseToolNamesFromUsedBy extracts a deduplicated list of tool names from the usedBy status.
func ParseToolNamesFromUsedBy(usedBy []AgentUsage) []string {
	toolSet := make(map[string]bool)
	for _, u := range usedBy {
		for _, t := range u.ToolNames {
			toolSet[t] = true
		}
	}
	var tools []string
	for t := range toolSet {
		tools = append(tools, t)
	}
	return tools
}

// Suppress unused import warning
var _ = strings.TrimSpace
