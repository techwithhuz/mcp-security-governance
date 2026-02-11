package discovery

import (
	"testing"
)

// ────────────────────────────────────────────────────────────────────────────
// getNestedMap
// ────────────────────────────────────────────────────────────────────────────

func TestGetNestedMap_SingleLevel(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"name": "test",
		},
	}

	result, ok := getNestedMap(obj, "spec")
	if !ok {
		t.Fatal("getNestedMap should succeed")
	}
	if result["name"] != "test" {
		t.Errorf("name = %v, want 'test'", result["name"])
	}
}

func TestGetNestedMap_MultiLevel(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"containers": "value",
			},
		},
	}

	result, ok := getNestedMap(obj, "spec", "template")
	if !ok {
		t.Fatal("getNestedMap multi-level should succeed")
	}
	if result["containers"] != "value" {
		t.Errorf("containers = %v, want 'value'", result["containers"])
	}
}

func TestGetNestedMap_MissingKey(t *testing.T) {
	obj := map[string]interface{}{
		"spec": map[string]interface{}{},
	}

	_, ok := getNestedMap(obj, "spec", "nonexistent")
	if ok {
		t.Error("getNestedMap with missing key should return false")
	}
}

func TestGetNestedMap_NotAMap(t *testing.T) {
	obj := map[string]interface{}{
		"spec": "not-a-map",
	}

	_, ok := getNestedMap(obj, "spec")
	if ok {
		t.Error("getNestedMap on non-map value should return false")
	}
}

func TestGetNestedMap_EmptyObject(t *testing.T) {
	obj := map[string]interface{}{}

	_, ok := getNestedMap(obj, "anything")
	if ok {
		t.Error("getNestedMap on empty object should return false")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// getNestedString
// ────────────────────────────────────────────────────────────────────────────

func TestGetNestedString_Found(t *testing.T) {
	obj := map[string]interface{}{
		"name": "hello",
	}

	val, ok := getNestedString(obj, "name")
	if !ok || val != "hello" {
		t.Errorf("getNestedString = (%q, %v), want ('hello', true)", val, ok)
	}
}

func TestGetNestedString_Missing(t *testing.T) {
	obj := map[string]interface{}{}

	_, ok := getNestedString(obj, "name")
	if ok {
		t.Error("getNestedString with missing key should return false")
	}
}

func TestGetNestedString_WrongType(t *testing.T) {
	obj := map[string]interface{}{
		"name": 42,
	}

	_, ok := getNestedString(obj, "name")
	if ok {
		t.Error("getNestedString with non-string value should return false")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// getNestedInt
// ────────────────────────────────────────────────────────────────────────────

func TestGetNestedInt_Int64(t *testing.T) {
	obj := map[string]interface{}{
		"port": int64(8080),
	}

	val, ok := getNestedInt(obj, "port")
	if !ok || val != 8080 {
		t.Errorf("getNestedInt(int64) = (%d, %v), want (8080, true)", val, ok)
	}
}

func TestGetNestedInt_Float64(t *testing.T) {
	// JSON unmarshaling produces float64 for numbers
	obj := map[string]interface{}{
		"port": float64(8080),
	}

	val, ok := getNestedInt(obj, "port")
	if !ok || val != 8080 {
		t.Errorf("getNestedInt(float64) = (%d, %v), want (8080, true)", val, ok)
	}
}

func TestGetNestedInt_Int(t *testing.T) {
	obj := map[string]interface{}{
		"port": int(8080),
	}

	val, ok := getNestedInt(obj, "port")
	if !ok || val != 8080 {
		t.Errorf("getNestedInt(int) = (%d, %v), want (8080, true)", val, ok)
	}
}

func TestGetNestedInt_Missing(t *testing.T) {
	obj := map[string]interface{}{}

	_, ok := getNestedInt(obj, "port")
	if ok {
		t.Error("getNestedInt with missing key should return false")
	}
}

func TestGetNestedInt_WrongType(t *testing.T) {
	obj := map[string]interface{}{
		"port": "not-a-number",
	}

	_, ok := getNestedInt(obj, "port")
	if ok {
		t.Error("getNestedInt with string value should return false")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// getNestedSlice
// ────────────────────────────────────────────────────────────────────────────

func TestGetNestedSlice_Found(t *testing.T) {
	obj := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
	}

	val, ok := getNestedSlice(obj, "items")
	if !ok {
		t.Fatal("getNestedSlice should succeed")
	}
	if len(val) != 3 {
		t.Errorf("len = %d, want 3", len(val))
	}
}

func TestGetNestedSlice_Missing(t *testing.T) {
	obj := map[string]interface{}{}

	_, ok := getNestedSlice(obj, "items")
	if ok {
		t.Error("getNestedSlice with missing key should return false")
	}
}

func TestGetNestedSlice_WrongType(t *testing.T) {
	obj := map[string]interface{}{
		"items": "not-a-slice",
	}

	_, ok := getNestedSlice(obj, "items")
	if ok {
		t.Error("getNestedSlice with non-slice value should return false")
	}
}

func TestGetNestedSlice_Empty(t *testing.T) {
	obj := map[string]interface{}{
		"items": []interface{}{},
	}

	val, ok := getNestedSlice(obj, "items")
	if !ok {
		t.Fatal("getNestedSlice on empty slice should succeed")
	}
	if len(val) != 0 {
		t.Errorf("len = %d, want 0", len(val))
	}
}
