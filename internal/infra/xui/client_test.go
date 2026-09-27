package xui_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"xui-sells-v2/internal/infra/xui"
)

const (
	testAPIKey = "secret-test-token-12345"
)

func newMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+testAPIKey {
			http.Error(w, `{"success":false,"msg":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}))
}

func TestAddClient(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/clients/add" {
			t.Errorf("expected path /panel/api/clients/add, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		var req xui.AddClientRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		if req.Client.Email != "user123_sub1" {
			t.Errorf("expected email user123_sub1, got %s", req.Client.Email)
		}
		if req.Client.LimitIP != 2 {
			t.Errorf("expected limitIp 2, got %d", req.Client.LimitIP)
		}
		if len(req.InboundIDs) != 2 || req.InboundIDs[0] != 1 || req.InboundIDs[1] != 2 {
			t.Errorf("unexpected inbound IDs: %v", req.InboundIDs)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"msg":"Client added","obj":{}}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)

	req := xui.AddClientRequest{
		Client: xui.ClientPayload{
			Email:      "user123_sub1",
			SubID:      "sub_id_16_chars",
			ID:         "550e8400-e29b-41d4-a716-446655440000",
			Flow:       "xtls-rprx-vision",
			TotalGB:    53687091200,
			ExpiryTime: 1735689600000,
			LimitIP:    2,
			TgID:       123456789,
			Group:      "vip_group",
			Comment:    "Sub ID: 42 | Plan: VIP",
			Enable:     true,
		},
		InboundIDs: []int{1, 2},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.AddClient(ctx, req); err != nil {
		t.Fatalf("AddClient failed: %v", err)
	}
}

func TestAddClientValidation(t *testing.T) {
	client := xui.NewClient("http://localhost:2053", testAPIKey)

	// Missing email
	err := client.AddClient(context.Background(), xui.AddClientRequest{
		Client:     xui.ClientPayload{Email: ""},
		InboundIDs: []int{1},
	})
	if err == nil {
		t.Fatalf("expected error for empty email")
	}

	// Missing inboundIDs
	err = client.AddClient(context.Background(), xui.AddClientRequest{
		Client:     xui.ClientPayload{Email: "test@example.com"},
		InboundIDs: []int{},
	})
	if err == nil {
		t.Fatalf("expected error for empty inboundIds")
	}
}

func TestUpdateClient(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/clients/update/user123_sub1" {
			t.Errorf("expected path /panel/api/clients/update/user123_sub1, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		var payload xui.ClientPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal client: %v", err)
		}

		if payload.LimitIP != 3 {
			t.Errorf("expected updated limitIp 3, got %d", payload.LimitIP)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"msg":"Client updated"}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)

	payload := xui.ClientPayload{
		Email:      "user123_sub1",
		SubID:      "rotated_sub_id",
		ID:         "550e8400-e29b-41d4-a716-446655440000",
		TotalGB:    53687091200,
		ExpiryTime: 1738368000000,
		LimitIP:    3,
		TgID:       123456789,
		Group:      "vip_group",
		Enable:     true,
	}

	if err := client.UpdateClient(context.Background(), "user123_sub1", payload); err != nil {
		t.Fatalf("UpdateClient failed: %v", err)
	}
}

func TestDeleteClient(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/clients/del/user123_sub1" {
			t.Errorf("expected path /panel/api/clients/del/user123_sub1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"msg":"Client deleted"}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)
	if err := client.DeleteClient(context.Background(), "user123_sub1"); err != nil {
		t.Fatalf("DeleteClient failed: %v", err)
	}
}

func TestGetClient(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/clients/get/user123_sub1" {
			t.Errorf("expected path /panel/api/clients/get/user123_sub1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"msg": "",
			"obj": {
				"client": {
					"email": "user123_sub1",
					"subId": "sub_xyz",
					"id": "uuid-v4",
					"limitIp": 2,
					"totalGB": 53687091200,
					"enable": true
				},
				"inboundIds": [1, 2],
				"traffic": {
					"up": 1024000,
					"down": 2048000,
					"total": 53687091200,
					"expiryTime": 1735689600000
				}
			}
		}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)
	res, err := client.GetClient(context.Background(), "user123_sub1")
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}

	if res.Client.Email != "user123_sub1" {
		t.Errorf("expected client email user123_sub1, got %s", res.Client.Email)
	}
	if len(res.InboundIDs) != 2 || res.InboundIDs[0] != 1 || res.InboundIDs[1] != 2 {
		t.Errorf("unexpected inbounds: %v", res.InboundIDs)
	}
	if res.Traffic.Up != 1024000 || res.Traffic.Down != 2048000 {
		t.Errorf("unexpected traffic stats: %+v", res.Traffic)
	}
}

func TestListInbounds(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/inbounds/list" {
			t.Errorf("expected path /panel/api/inbounds/list, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"msg": "",
			"obj": [
				{
					"id": 1,
					"port": 443,
					"protocol": "vless",
					"tag": "inbound-443",
					"remark": "VLESS-Reality",
					"enable": true,
					"up": 1024,
					"down": 2048,
					"total": 10737418240,
					"clientStats": [
						{
							"id": 10,
							"inboundId": 1,
							"email": "user1",
							"subId": "sub1",
							"uuid": "uuid1",
							"up": 512,
							"down": 1024,
							"total": 53687091200,
							"enable": true
						}
					]
				}
			]
		}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)
	inbounds, err := client.ListInbounds(context.Background())
	if err != nil {
		t.Fatalf("ListInbounds failed: %v", err)
	}

	if len(inbounds) != 1 {
		t.Fatalf("expected 1 inbound, got %d", len(inbounds))
	}
	if inbounds[0].ID != 1 || inbounds[0].Port != 443 || inbounds[0].Protocol != "vless" {
		t.Errorf("unexpected inbound fields: %+v", inbounds[0])
	}
	if len(inbounds[0].ClientStats) != 1 || inbounds[0].ClientStats[0].Email != "user1" {
		t.Errorf("unexpected clientStats: %+v", inbounds[0].ClientStats)
	}
}

func TestGetSubLinks(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/clients/subLinks/test_sub_123" {
			t.Errorf("expected path /panel/api/clients/subLinks/test_sub_123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"msg": "",
			"obj": [
				"vless://uuid@host:443?security=reality#user1",
				"vmess://eyJ2IjoyLC..."
			]
		}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)
	links, err := client.GetSubLinks(context.Background(), "test_sub_123")
	if err != nil {
		t.Fatalf("GetSubLinks failed: %v", err)
	}

	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if !strings.HasPrefix(links[0], "vless://") {
		t.Errorf("expected vless link, got %s", links[0])
	}
}

func TestResetTraffic(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/clients/resetTraffic/user123_sub1" {
			t.Errorf("expected path /panel/api/clients/resetTraffic/user123_sub1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"msg":"Traffic reset"}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)
	if err := client.ResetTraffic(context.Background(), "user123_sub1"); err != nil {
		t.Fatalf("ResetTraffic failed: %v", err)
	}
}

func TestGetSettings(t *testing.T) {
	ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/panel/api/setting/all" {
			t.Errorf("expected path /panel/api/setting/all, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"msg": "",
			"obj": {
				"subURI": "https://sub.myvpn.com:8443/sub/",
				"subPath": "/sub/",
				"webPort": 2053
			}
		}`))
	})
	defer ts.Close()

	client := xui.NewClient(ts.URL, testAPIKey)
	settings, err := client.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}

	if settings.SubURI != "https://sub.myvpn.com:8443/sub/" {
		t.Errorf("expected subURI https://sub.myvpn.com:8443/sub/, got %s", settings.SubURI)
	}
	if settings.SubPath != "/sub/" {
		t.Errorf("expected subPath /sub/, got %s", settings.SubPath)
	}
}

func TestBuildSubscriptionURL(t *testing.T) {
	tests := []struct {
		name     string
		panelURL string
		subURI   string
		subPath  string
		subId    string
		expected string
	}{
		{
			name:     "Populated subURI with trailing slash",
			panelURL: "http://127.0.0.1:2053",
			subURI:   "https://sub.myvpn.com:8443/sub/",
			subPath:  "/sub/",
			subId:    "random_sub_id_42",
			expected: "https://sub.myvpn.com:8443/sub/random_sub_id_42",
		},
		{
			name:     "Populated subURI without trailing slash",
			panelURL: "http://127.0.0.1:2053",
			subURI:   "https://sub.myvpn.com:8443/sub",
			subPath:  "/sub/",
			subId:    "random_sub_id_42",
			expected: "https://sub.myvpn.com:8443/sub/random_sub_id_42",
		},
		{
			name:     "Empty subURI with standard subPath",
			panelURL: "http://127.0.0.1:2053",
			subURI:   "",
			subPath:  "/sub/",
			subId:    "user_sub_id",
			expected: "http://127.0.0.1:2053/sub/user_sub_id",
		},
		{
			name:     "Empty subURI with panelURL trailing slash and subPath without slashes",
			panelURL: "https://panel.example.com:2053/",
			subURI:   "",
			subPath:  "sub",
			subId:    "user_sub_id",
			expected: "https://panel.example.com:2053/sub/user_sub_id",
		},
		{
			name:     "Empty subURI and empty subPath defaults to /sub/",
			panelURL: "https://panel.example.com:2053",
			subURI:   "",
			subPath:  "",
			subId:    "user_sub_id",
			expected: "https://panel.example.com:2053/sub/user_sub_id",
		},
		{
			name:     "Custom subPath with custom prefix",
			panelURL: "https://panel.example.com:2053",
			subURI:   "",
			subPath:  "/custom-sub/",
			subId:    "xyz",
			expected: "https://panel.example.com:2053/custom-sub/xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xui.BuildSubscriptionURL(tt.panelURL, tt.subURI, tt.subPath, tt.subId)
			if got != tt.expected {
				t.Errorf("BuildSubscriptionURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	t.Run("Unauthorized response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"success":false,"msg":"Unauthorized"}`, http.StatusUnauthorized)
		}))
		defer ts.Close()

		client := xui.NewClient(ts.URL, "wrong-token")
		err := client.DeleteClient(context.Background(), "user1")
		if err == nil || !strings.Contains(err.Error(), "unauthorized") {
			t.Fatalf("expected unauthorized error, got %v", err)
		}
	})

	t.Run("Server 500 error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}))
		defer ts.Close()

		client := xui.NewClient(ts.URL, testAPIKey)
		err := client.DeleteClient(context.Background(), "user1")
		if err == nil || !strings.Contains(err.Error(), "status 500") {
			t.Fatalf("expected status 500 error, got %v", err)
		}
	})

	t.Run("API logical failure (success: false)", func(t *testing.T) {
		ts := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":false,"msg":"Client already exists"}`))
		})
		defer ts.Close()

		client := xui.NewClient(ts.URL, testAPIKey)
		err := client.AddClient(context.Background(), xui.AddClientRequest{
			Client:     xui.ClientPayload{Email: "user1"},
			InboundIDs: []int{1},
		})
		if err == nil || !strings.Contains(err.Error(), "Client already exists") {
			t.Fatalf("expected 'Client already exists' error, got %v", err)
		}
	})
}
