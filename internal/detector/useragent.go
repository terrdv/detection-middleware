package detector

import (
	"net/http"
	"strings"

	"detection-middleware/internal/store"
)

// botUAPatterns are substrings that appear in the User-Agent of common
// scripting tools, libraries, and crawlers. Matched case-insensitively.
var botUAPatterns = []string{
	"curl",
	"wget",
	"python-requests",
	"python-urllib",
	"go-http-client",
	"java/",
	"okhttp",
	"scrapy",
	"httpclient",
	"headlesschrome",
	"phantomjs",
	"puppeteer",
	"playwright",
	"selenium",
	"bot",
	"crawler",
	"spider",
}

type userAgentSignal struct{}

func (userAgentSignal) Name() string { return "user_agent" }

func (userAgentSignal) Score(r *http.Request, _ *store.ClientState) float64 {
	ua := strings.TrimSpace(r.UserAgent())
	if ua == "" {
		return 1
	}
	lower := strings.ToLower(ua)
	for _, p := range botUAPatterns {
		if strings.Contains(lower, p) {
			return 1
		}
	}

	if strings.HasPrefix(ua, "Mozilla/") {
		return 0
	}

	return 0.4
}
