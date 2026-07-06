package channels

import "testing"

func TestResolveWebhookBaseURL(t *testing.T) {
	tests := []struct {
		name       string
		request    string
		configured string
		want       string
	}{
		{
			name:       "configured https wins",
			request:    "http://localhost:8080",
			configured: "https://bot.example.com",
			want:       "https://bot.example.com",
		},
		{
			name:       "request https used when configured empty",
			request:    "https://admin.example.com",
			configured: "",
			want:       "https://admin.example.com",
		},
		{
			name:       "http request without configured is rejected",
			request:    "http://localhost:8080",
			configured: "",
			want:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveWebhookBaseURL(tt.request, tt.configured); got != tt.want {
				t.Fatalf("ResolveWebhookBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
