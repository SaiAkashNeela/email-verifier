package validator

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const disposableDomainsURL = "https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/refs/heads/main/disposable_email_blocklist.conf"

// ExternalDisposableChecker checks if a domain is in an external disposable email list.
type ExternalDisposableChecker struct {
	disposableDomains map[string]struct{}
	mu                sync.RWMutex
	lastUpdated       time.Time
}

// NewExternalDisposableChecker creates a new instance of ExternalDisposableChecker.
func NewExternalDisposableChecker() *ExternalDisposableChecker {
	return &ExternalDisposableChecker{
		disposableDomains: make(map[string]struct{}),
	}
}

// LoadDisposableDomains fetches the disposable email domains from the external URL.
func (c *ExternalDisposableChecker) LoadDisposableDomains() error {
	log.Printf("Fetching disposable domains from %s", disposableDomainsURL)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(disposableDomainsURL)
	if err != nil {
		return fmt.Errorf("failed to fetch disposable domains list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch disposable domains list: received status code %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	newDomains := make(map[string]struct{})
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain != "" && !strings.HasPrefix(domain, "#") { // Ignore comments
			newDomains[strings.ToLower(domain)] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read disposable domains list: %w", err)
	}

	c.mu.Lock()
	c.disposableDomains = newDomains
	c.lastUpdated = time.Now()
	c.mu.Unlock()

	return nil
}

// IsDisposable checks if the given domain is in the loaded disposable domains list.
func (c *ExternalDisposableChecker) IsDisposable(domain string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, found := c.disposableDomains[strings.ToLower(domain)]
	return found
}

// GetLastUpdated returns the time when the disposable domains list was last updated.
func (c *ExternalDisposableChecker) GetLastUpdated() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdated
}