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

const disposableBlocklistURL = "https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/refs/heads/main/disposable_email_blocklist.conf"

// DisposableBlocklist manages the loading and checking of disposable email domains.
type DisposableBlocklist struct {
	domains map[string]struct{}
	once    sync.Once
	mu      sync.RWMutex // Protects access to the domains map
}

// NewDisposableBlocklist creates and returns a new DisposableBlocklist instance.
func NewDisposableBlocklist() *DisposableBlocklist {
	return &DisposableBlocklist{
		domains: make(map[string]struct{}),
	}
}

// Load fetches the disposable email domain blocklist from the URL and populates the internal map.
// It uses sync.Once to ensure the list is loaded only once.
func (db *DisposableBlocklist) Load() error {
	var err error
	db.once.Do(func() {
		log.Println("Loading disposable email domain blocklist...")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, httpErr := client.Get(disposableBlocklistURL)
		if httpErr != nil {
			err = fmt.Errorf("failed to fetch disposable domains: %w", httpErr)
			log.Printf("Error fetching disposable domains: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("failed to fetch disposable domains, status code: %d", resp.StatusCode)
			log.Printf("Error fetching disposable domains: %v", err)
			return
		}

		newDomains := make(map[string]struct{})
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			domain := strings.TrimSpace(scanner.Text())
			if domain != "" && !strings.HasPrefix(domain, "#") { // Ignore empty lines and comments
				newDomains[domain] = struct{}{}
			}
		}

		if scanErr := scanner.Err(); scanErr != nil {
			err = fmt.Errorf("failed to read disposable domains: %w", scanErr)
			log.Printf("Error reading disposable domains: %v", err)
			return
		}

		db.mu.Lock()
		db.domains = newDomains
		db.mu.Unlock()
		log.Printf("Successfully loaded %d disposable email domains.", len(newDomains))
	})
	return err
}

// IsDisposable checks if the given domain is present in the disposable email domain blocklist.
func (db *DisposableBlocklist) IsDisposable(domain string) bool {
	// Ensure the list is loaded before checking
	if err := db.Load(); err != nil {
		log.Printf("Warning: Disposable blocklist not loaded, cannot check domain %s: %v", domain, err)
		return false // Cannot confirm, so assume not disposable
	}

	db.mu.RLock()
	_, found := db.domains[strings.ToLower(domain)]
	db.mu.RUnlock()
	return found
}