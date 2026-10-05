package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestValidTimeZonesPaginatesWithLimit50(t *testing.T) {
	var queries []string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2026/org-config/valid-time-zones" || r.Header.Get("X-SailPoint-Experimental") != "true" {
			t.Errorf("unexpected request %s %v", r.URL.Path, r.Header)
		}
		queries = append(queries, r.URL.RawQuery)
		page := []string{}
		if r.URL.Query().Get("offset") == "0" {
			for i := 0; i < validTimeZonesPageSize; i++ {
				page = append(page, fmt.Sprintf("Zone/%d", i))
			}
		} else {
			page = append(page, "Europe/Warsaw")
		}
		_ = json.NewEncoder(w).Encode(page)
	})
	client := NewClient(context.Background(), server.URL, "id", "secret", 1000)
	zones, err := client.GetValidTimeZones(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(zones) != validTimeZonesPageSize+1 || zones[validTimeZonesPageSize] != "Europe/Warsaw" {
		t.Fatalf("unexpected zones %v", zones)
	}
	if len(queries) != 2 || queries[0] != "limit=50&offset=0" || queries[1] != "limit=50&offset=50" {
		t.Fatalf("unexpected queries %v", queries)
	}
}
