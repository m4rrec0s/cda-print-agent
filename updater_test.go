package main

import (
	"net/url"
	"testing"
)

func TestIsSecureUpdateURL(t *testing.T) {
	cases := []struct {
		url      string
		wantSafe bool
	}{
		{"https://updates.example.com/agent.exe", true},
		{"http://localhost:3333/agent.exe", true},
		{"http://127.0.0.1/agent.exe", true},
		{"http://updates.example.com/agent.exe", false},
		{"file:///tmp/agent.exe", false},
	}
	for _, testCase := range cases {
		t.Run(testCase.url, func(t *testing.T) {
			parsed, err := url.Parse(testCase.url)
			if err != nil {
				t.Fatalf("parse URL: %v", err)
			}
			if got := isSecureUpdateURL(parsed); got != testCase.wantSafe {
				t.Fatalf("isSecureUpdateURL(%q) = %t, want %t", testCase.url, got, testCase.wantSafe)
			}
		})
	}
}
