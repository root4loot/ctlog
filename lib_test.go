package crtsher

import (
	"net/http"
	"testing"
	"time"
)

func TestNewRunner(t *testing.T) {
	runner := NewRunner()
	if runner.Options.Concurrency != 3 {
		t.Errorf("Expected Concurrency to be 3, got %d", runner.Options.Concurrency)
	}
	if runner.Options.Timeout != 90 {
		t.Errorf("Expected Timeout to be 90, got %d", runner.Options.Timeout)
	}
}

func TestQuery(t *testing.T) {
	runner := NewRunnerWithOptions(&Options{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	})

	domains := []string{"example.com", "hackerone.com"}
	success := false

	for _, domain := range domains {
		results := runner.Query(domain)
		if len(results) > 0 {
			success = true
			break
		}
	}

	if !success {
		t.Error("Expected results for at least one domain, got none")
	}
}

func TestGetCommonName(t *testing.T) {
	result := Result{CommonName: "*.example.com"}
	expected := "example.com"
	if result.GetCommonName() != expected {
		t.Errorf("Expected %s, got %s", expected, result.GetCommonName())
	}
}

func TestGetMatchingIdentity(t *testing.T) {
	result := Result{NameValue: "*.example.com"}
	expected := "example.com"
	if result.GetMatchingIdentity() != expected {
		t.Errorf("Expected %s, got %s", expected, result.GetMatchingIdentity())
	}
}
