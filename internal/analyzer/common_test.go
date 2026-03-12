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

func TestGetServiceNameEnriched(t *testing.T) {
	defaultLabels := []string{
		"service_name", "service", "app", "application", "name",
		"app_kubernetes_io_name", "k8s.container.name", "k8s.deployment.name",
		"k8s.pod.name", "container", "component", "workload", "job",
	}

	tests := []struct {
		name   string
		attrs  map[string]string
		labels []string
		want   string
	}{
		{
			name:   "service.name takes priority over label list",
			attrs:  map[string]string{"service.name": "payments", "app": "frontend"},
			labels: defaultLabels,
			want:   "payments",
		},
		{
			name:   "resolves from k8s.container.name when no service.name",
			attrs:  map[string]string{"k8s.container.name": "nova-powerplay-app"},
			labels: defaultLabels,
			want:   "nova-powerplay-app",
		},
		{
			name:   "priority list respected – app before k8s.container.name",
			attrs:  map[string]string{"app": "frontend", "k8s.container.name": "frontend-container"},
			labels: defaultLabels,
			want:   "frontend",
		},
		{
			name:   "first-match wins with custom labels",
			attrs:  map[string]string{"my_app": "payments", "service": "ignored"},
			labels: []string{"my_app", "service"},
			want:   "payments",
		},
		{
			name:   "no match falls back to unknown_service",
			attrs:  map[string]string{"host.name": "worker-1"},
			labels: defaultLabels,
			want:   "unknown_service",
		},
		{
			name:   "empty labels list falls back to service.name-only behaviour (unknown)",
			attrs:  map[string]string{"host.name": "worker-1"},
			labels: []string{},
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getServiceName(tt.attrs, tt.labels...); got != tt.want {
				t.Fatalf("getServiceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

