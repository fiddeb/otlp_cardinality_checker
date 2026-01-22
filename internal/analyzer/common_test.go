package analyzer

import "testing"

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{
			name:  "uses service.name when present",
			attrs: map[string]string{"service.name": "checkout"},
			want:  "checkout",
		},
		{
			name:  "defaults to unknown when service.name missing",
			attrs: map[string]string{"host.name": "host-1"},
			want:  "unknown",
		},
		{
			name:  "defaults to unknown when service.name empty",
			attrs: map[string]string{"service.name": ""},
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getServiceName(tt.attrs); got != tt.want {
				t.Fatalf("getServiceName() = %q, want %q", got, tt.want)
			}
		})
	}
}
