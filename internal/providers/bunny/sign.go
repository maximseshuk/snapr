package bunny

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// timestampLayout matches LastChanged/DateCreated in Bunny list responses.
const timestampLayout = "2006-01-02T15:04:05.999"

func ParseTimestamp(s string) (time.Time, error) {
	return time.Parse(timestampLayout, s)
}

// SignOptions configures Bunny Pull Zone Advanced Token Authentication (HMAC-SHA256).
type SignOptions struct {
	Hostname    string // Pull Zone hostname (e.g. "myzone.b-cdn.net")
	SecurityKey string // Token Authentication security key from the dashboard
	Path        string // absolute URL path of the requested file (must start with /)
	TTL         int    // token lifetime in seconds; defaults to 3600
}

func SignURL(opts SignOptions) string {
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 3600
	}
	expires := time.Now().Unix() + int64(ttl)

	mac := hmac.New(sha256.New, []byte(opts.SecurityKey))
	mac.Write([]byte(opts.Path + strconv.FormatInt(expires, 10)))
	token := "HS256-" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	q := url.Values{}
	q.Set("token", token)
	q.Set("expires", strconv.FormatInt(expires, 10))

	return "https://" + normalizeHost(opts.Hostname) + opts.Path + "?" + q.Encode()
}

func normalizeHost(hostname string) string {
	hostname = strings.TrimSuffix(hostname, "/")
	hostname = strings.TrimPrefix(hostname, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")
	return hostname
}
