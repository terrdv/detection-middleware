package detector

import (
	"net/http/httptest"
	"testing"

	"detection-middleware/internal/config"
)

func TestClientKey(t *testing.T) {
	tests := []struct {
		name       string
		trustXFF   bool
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "strips port from IPv4",
			remoteAddr: "203.0.113.7:54321",
			want:       "203.0.113.7",
		},
		{
			name:       "strips port from IPv6",
			remoteAddr: "[2001:db8::1]:443",
			want:       "2001:db8::1",
		},
		{
			name:       "no port falls back to raw value",
			remoteAddr: "203.0.113.7",
			want:       "203.0.113.7",
		},
		{
			name:       "ignores XFF when not trusted",
			trustXFF:   false,
			remoteAddr: "203.0.113.7:54321",
			xff:        "10.0.0.9",
			want:       "203.0.113.7",
		},
		{
			name:       "uses XFF when trusted",
			trustXFF:   true,
			remoteAddr: "203.0.113.7:54321",
			xff:        "10.0.0.9",
			want:       "10.0.0.9",
		},
		{
			name:       "takes first hop of XFF list and trims",
			trustXFF:   true,
			remoteAddr: "203.0.113.7:54321",
			xff:        " 10.0.0.9 , 172.16.0.1, 192.168.1.1",
			want:       "10.0.0.9",
		},
		{
			name:       "trusted but no XFF falls back to source IP",
			trustXFF:   true,
			remoteAddr: "203.0.113.7:54321",
			xff:        "",
			want:       "203.0.113.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Detector{cfg: config.Config{TrustForwardedFor: tt.trustXFF}}

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			if got := d.clientKey(req); got != tt.want {
				t.Errorf("clientKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
