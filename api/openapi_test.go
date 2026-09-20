package api_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

const contractPath = "openapi.yaml"

func loadContract(t *testing.T) *openapi3.T {
	t.Helper()

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	doc, err := loader.LoadFromFile(contractPath)
	if err != nil {
		t.Fatalf("load OpenAPI contract: %v", err)
	}

	return doc
}

func TestOpenAPIContractIsValid(t *testing.T) {
	doc := loadContract(t)

	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version = %q, want 3.1.0", doc.OpenAPI)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI contract: %v", err)
	}
}

func TestOpenAPIPathsAndOperationIDs(t *testing.T) {
	doc := loadContract(t)
	seenOperationIDs := make(map[string]string)

	for path, item := range doc.Paths.Map() {
		if !strings.HasPrefix(path, "/api/v2/") {
			t.Errorf("path %q must start with /api/v2/", path)
		}
		if strings.HasSuffix(path, "/") {
			t.Errorf("path %q must not end with a slash", path)
		}
		if strings.HasSuffix(path, "/enable") || strings.HasSuffix(path, "/disable") {
			t.Errorf("path %q uses an RPC action; expose state through PATCH", path)
		}

		for method, operation := range item.Operations() {
			location := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
			if operation.OperationID == "" {
				t.Errorf("%s has no operationId", location)
				continue
			}
			if previous, exists := seenOperationIDs[operation.OperationID]; exists {
				t.Errorf("operationId %q is used by both %s and %s", operation.OperationID, previous, location)
			}
			seenOperationIDs[operation.OperationID] = location

			if method == "DELETE" && operation.RequestBody != nil {
				t.Errorf("%s must identify the relationship in the URL instead of using a DELETE body", location)
			}
		}
	}
}

func TestOpenAPIMonitorCRUDAndPartialUpdate(t *testing.T) {
	doc := loadContract(t)
	collection := doc.Paths.Find("/api/v2/monitors")
	item := doc.Paths.Find("/api/v2/monitors/{monitorId}")

	if collection == nil || collection.Get == nil || collection.Post == nil {
		t.Fatal("monitor collection must support GET and POST")
	}
	if item == nil || item.Get == nil || item.Patch == nil || item.Delete == nil {
		t.Fatal("monitor resource must support GET, PATCH and DELETE")
	}

	media := item.Patch.RequestBody.Value.Content.Get("application/json")
	if media == nil || media.Schema == nil || media.Schema.Ref != "#/components/schemas/MonitorPatch" {
		t.Fatal("monitor PATCH must use the MonitorPatch schema")
	}
	if media.Schema.Value != nil && len(media.Schema.Value.Required) != 0 {
		t.Fatal("MonitorPatch must not require fields that are absent from a partial update")
	}
}

func TestOpenAPIExamplesArePresentAndValid(t *testing.T) {
	doc := loadContract(t)
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("examples must conform to their schemas: %v", err)
	}

	exampleCount := 0
	for _, item := range doc.Paths.Map() {
		for _, operation := range item.Operations() {
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				for _, media := range operation.RequestBody.Value.Content {
					if media.Example != nil {
						exampleCount++
					}
					exampleCount += len(media.Examples)
				}
			}
			for _, response := range operation.Responses.Map() {
				if response == nil || response.Value == nil {
					continue
				}
				for _, media := range response.Value.Content {
					if media.Example != nil {
						exampleCount++
					}
					exampleCount += len(media.Examples)
				}
			}
		}
	}

	if exampleCount < 12 {
		t.Fatalf("validated examples = %d, want at least 12 request/response examples", exampleCount)
	}
}
