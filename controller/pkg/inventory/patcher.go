package inventory

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// PublisherVerification represents the publisher trust verification information.
// This mirrors the structure in the CRD status.publisher field.
type PublisherVerification struct {
	VerifiedPublisher bool   `json:"verifiedPublisher"`
	VerifiedOrganization bool `json:"verifiedOrganization"`
	Score int `json:"score"`
	Grade string `json:"grade"`
	GradedAt metav1.Time `json:"gradedAt"`
}

// StatusPatcher patches MCPServerCatalog resources with governance scores.
type StatusPatcher struct {
	dynClient dynamic.Interface
	gvr       schema.GroupVersionResource
}

// NewStatusPatcher creates a new status patcher.
func NewStatusPatcher(client dynamic.Interface) *StatusPatcher {
	return &StatusPatcher{
		dynClient: client,
		gvr: MCPServerCatalogGVR,
	}
}

// PatchCatalogStatus patches the .status.publisher field of an MCPServerCatalog
// with the verified score information from the governance controller.
func (p *StatusPatcher) PatchCatalogStatus(ctx context.Context, resource *VerifiedResource) error {
	if resource == nil {
		return fmt.Errorf("resource cannot be nil")
	}

	// Build the publisher verification object
	pubVerif := PublisherVerification{
		VerifiedPublisher: resource.VerifiedScore.PublisherScore >= 70,
		VerifiedOrganization: resource.VerifiedScore.OrgScore >= 70,
		Score: resource.VerifiedScore.Score,
		Grade: resource.VerifiedScore.Grade,
		GradedAt: metav1.NewTime(time.Now()),
	}

	// Construct the patch JSON
	patchPayload := map[string]interface{}{
		"status": map[string]interface{}{
			"publisher": pubVerif,
		},
	}

	// Convert to JSON bytes
	patchBytes, err := marshalPatch(patchPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}

	// Apply the patch using the status subresource
	resourceClient := p.dynClient.Resource(p.gvr).Namespace(resource.Namespace)
	_, err = resourceClient.Patch(
		ctx,
		resource.Name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
		"status",
	)
	if err != nil {
		return fmt.Errorf("failed to patch %s/%s status: %w", resource.Namespace, resource.Name, err)
	}

	log.Printf("[patcher] Successfully patched %s/%s status: score=%d, grade=%s, verifiedPublisher=%v, verifiedOrg=%v",
		resource.Namespace, resource.Name, pubVerif.Score, pubVerif.Grade, 
		pubVerif.VerifiedPublisher, pubVerif.VerifiedOrganization)

	return nil
}

// PatchMultipleCatalogs patches multiple catalog resources in parallel.
// Returns the number of successful patches and any error encountered.
func (p *StatusPatcher) PatchMultipleCatalogs(ctx context.Context, resources []*VerifiedResource) (int, error) {
	if len(resources) == 0 {
		return 0, nil
	}

	// Create a channel for results
	type result struct {
		success bool
		err     error
	}
	results := make(chan result, len(resources))

	// Patch in parallel (limited concurrency to avoid overwhelming the API)
	concurrency := 5
	semaphore := make(chan struct{}, concurrency)

	for _, res := range resources {
		go func(r *VerifiedResource) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := p.PatchCatalogStatus(ctx, r)
			results <- result{success: err == nil, err: err}
		}(res)
	}

	// Collect results
	successCount := 0
	var lastErr error
	for i := 0; i < len(resources); i++ {
		res := <-results
		if res.success {
			successCount++
		} else if res.err != nil {
			lastErr = res.err
			log.Printf("[patcher] WARNING: %v", res.err)
		}
	}

	if lastErr != nil && successCount < len(resources) {
		return successCount, fmt.Errorf("patched %d/%d catalogs, last error: %w", successCount, len(resources), lastErr)
	}

	return successCount, nil
}

// marshalPatch converts a patch object to JSON bytes.
func marshalPatch(patch interface{}) ([]byte, error) {
	// Use unstructured for simple JSON marshaling
	u := &unstructured.Unstructured{Object: make(map[string]interface{})}
	
	// Convert interface{} to unstructured object
	data, err := convertToMap(patch)
	if err != nil {
		return nil, err
	}
	
	u.Object = data.(map[string]interface{})
	return u.MarshalJSON()
}

// convertToMap recursively converts structs to maps for JSON serialization.
func convertToMap(obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			converted, err := convertToMap(val)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			converted, err := convertToMap(val)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	case PublisherVerification:
		return map[string]interface{}{
			"verifiedPublisher": v.VerifiedPublisher,
			"verifiedOrganization": v.VerifiedOrganization,
			"score": v.Score,
			"grade": v.Grade,
			"gradedAt": v.GradedAt,
		}, nil
	case metav1.Time:
		return v.Format(time.RFC3339), nil
	default:
		return v, nil
	}
}
