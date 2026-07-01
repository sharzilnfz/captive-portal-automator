package prober

import (
	"context"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// ProbeResult represents the outcome of a connectivity check.
type ProbeResult struct {
	Online    bool
	PortalURL string
	HTML      string
	FinalURL  string
}

// Prober checks internet connectivity against known endpoints.
type Prober struct {
	client    *http.Client
	endpoints []Endpoint
	logger    *slog.Logger
}

// NewProber creates a Prober with the given HTTP client and logger.
func NewProber(client *http.Client, logger *slog.Logger) *Prober {
	return &Prober{
		client:    client,
		endpoints: DefaultEndpoints(),
		logger:    logger,
	}
}

// SetEndpoints replaces the probe endpoints (useful for testing).
func (p *Prober) SetEndpoints(eps []Endpoint) {
	p.endpoints = eps
}

// Check determines if the device is online or behind a captive portal.
func (p *Prober) Check(ctx context.Context) ProbeResult {
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := backoff(attempt)
			p.logger.Debug("retrying probe", "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return ProbeResult{Online: false}
			case <-time.After(delay):
			}
		}

		endCount := minInt(2+attempt, len(p.endpoints))
		for i := 0; i < endCount; i++ {
			result := p.probeEndpoint(ctx, p.endpoints[i])
			if result.Online || result.PortalURL != "" {
				return result
			}
		}
	}

	return ProbeResult{Online: false}
}

func (p *Prober) probeEndpoint(ctx context.Context, ep Endpoint) ProbeResult {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ep.URL, nil)
	if err != nil {
		p.logger.Debug("probe request error", "url", ep.URL, "error", err)
		return ProbeResult{Online: false, FinalURL: ep.URL}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Debug("probe fetch error", "url", ep.URL, "error", err)
		return ProbeResult{Online: false, FinalURL: ep.URL}
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	body := string(bodyBytes)
	finalURL := resp.Request.URL.String()
	redirected := finalURL != ep.URL

	if ep.CheckFunc(resp.StatusCode, body, redirected) {
		p.logger.Debug("probe online", "url", ep.URL)
		return ProbeResult{Online: true, FinalURL: finalURL}
	}

	if redirected {
		p.logger.Info("portal redirect detected", "from", ep.URL, "to", finalURL)
		return ProbeResult{Online: false, PortalURL: finalURL, HTML: body, FinalURL: finalURL}
	}

	if portalURL := ExtractPortalURL(body, ep.URL); portalURL != "" {
		p.logger.Info("portal URL extracted from HTML", "url", portalURL)
		return ProbeResult{Online: false, PortalURL: portalURL, HTML: body, FinalURL: finalURL}
	}

	p.logger.Debug("probe inconclusive", "url", ep.URL, "status", resp.StatusCode)
	return ProbeResult{Online: false, HTML: body, FinalURL: finalURL}
}

// ExtractPortalURL tries to find the portal login URL from intercepted HTML.
func ExtractPortalURL(html, baseURL string) string {
	if html == "" {
		return ""
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	// 1. <meta http-equiv="refresh" content="0;url=...">
	metaRe := regexp.MustCompile(`(?i)<meta[^>]+http-equiv\s*=\s*["']?refresh["']?[^>]+content\s*=\s*["'][^"']*?url\s*=\s*([^"'\s>]+)`)
	if m := metaRe.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	// reversed attribute order
	metaRe2 := regexp.MustCompile(`(?i)<meta[^>]+content\s*=\s*["'][^"']*?url\s*=\s*([^"'\s>]+)[^>]*http-equiv\s*=\s*["']?refresh["']?`)
	if m := metaRe2.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	// 2. <form action="...">
	formRe := regexp.MustCompile(`(?i)<form[^>]+action\s*=\s*["']([^"']+)["']`)
	if m := formRe.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	// 3. First <a href> pointing to a different host
	linkRe := regexp.MustCompile(`(?i)<a[^>]+href\s*=\s*["']([^"'#][^"']*)["']`)
	for _, m := range linkRe.FindAllStringSubmatch(html, -1) {
		if u, err := base.Parse(m[1]); err == nil {
			if (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != base.Hostname() {
				return u.String()
			}
		}
	}

	// 4. JS redirect: location.href = "..." or location.replace("...")
	jsRe := regexp.MustCompile(`(?i)location(?:\.href|\.replace)\s*=\s*["']([^"']+)["']`)
	if m := jsRe.FindStringSubmatch(html); m != nil {
		if u, err := base.Parse(m[1]); err == nil {
			return u.String()
		}
	}

	return ""
}

func backoff(attempt int) time.Duration {
	base := 500 * time.Millisecond
	d := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	jitter := time.Duration(rand.Int63n(int64(d) / 2))
	d = d - time.Duration(int64(d)/4) + jitter
	if d > 8*time.Second {
		d = 8 * time.Second
	}
	return d
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
