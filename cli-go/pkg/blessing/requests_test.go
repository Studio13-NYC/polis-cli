package blessing

import "testing"

func TestDomainMatches(t *testing.T) {
	tests := []struct {
		url    string
		domain string
		want   bool
	}{
		{"https://testpilot.polis.pub/posts/20260101/hello.md", "testpilot.polis.pub", true},
		{"https://follower1.polis.pub/comments/20260101/reply.md", "testpilot.polis.pub", false},
		{"https://follower1.polis.pub/comments/20260101/reply.md", "follower1.polis.pub", true},
		{"https://TESTPILOT.POLIS.PUB/posts/test.md", "testpilot.polis.pub", true}, // case-insensitive
		{"not-a-url", "testpilot.polis.pub", false},
		{"", "testpilot.polis.pub", false},
	}

	for _, tt := range tests {
		got := domainMatches(tt.url, tt.domain)
		if got != tt.want {
			t.Errorf("domainMatches(%q, %q) = %v, want %v", tt.url, tt.domain, got, tt.want)
		}
	}
}
