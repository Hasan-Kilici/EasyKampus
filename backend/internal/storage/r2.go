package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"
)

// R2Config holds the credentials required to sign S3-compatible requests
// against Cloudflare R2. R2 uses the "auto" region.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
	PublicURL       string
}

func (c R2Config) Configured() bool {
	return c.AccessKeyID != "" && c.SecretAccessKey != "" && c.Bucket != ""
}

func (c R2Config) host() string {
	if c.Endpoint != "" {
		if parsed, err := url.Parse(c.Endpoint); err == nil && parsed.Host != "" {
			return parsed.Host
		}
		return strings.TrimPrefix(strings.TrimPrefix(c.Endpoint, "https://"), "http://")
	}
	return c.AccountID + ".r2.cloudflarestorage.com"
}

// PresignPut returns a presigned URL that allows a direct HTTP PUT of an
// object to the bucket. The caller must send exactly the same Content-Type.
func (c R2Config) PresignPut(key string, contentType string, expiry time.Duration) (string, error) {
	if !c.Configured() {
		return "", errors.New("r2 is not configured")
	}
	return c.presign("PUT", key, contentType, expiry)
}

// PresignGet returns a presigned URL for reading a private object.
func (c R2Config) PresignGet(key string, expiry time.Duration) (string, error) {
	if !c.Configured() {
		return "", errors.New("r2 is not configured")
	}
	return c.presign("GET", key, "", expiry)
}

// PublicURL returns the public object URL when a custom domain is configured.
func (c R2Config) PublicURLFor(key string) string {
	if c.PublicURL == "" {
		return ""
	}
	return c.PublicURL + "/" + strings.TrimLeft(key, "/")
}

func (c R2Config) presign(method string, key string, contentType string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	if expiry > 7*24*time.Hour {
		expiry = 7 * 24 * time.Hour
	}
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	region := "auto"
	service := "s3"
	host := c.host()
	scope := dateStamp + "/" + region + "/" + service + "/aws4_request"

	canonicalURI := "/" + c.Bucket + "/" + encodePath(key)
	query := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    c.AccessKeyID + "/" + scope,
		"X-Amz-Date":          amzDate,
		"X-Amz-Expires":       strconvItoa(int(expiry / time.Second)),
		"X-Amz-SignedHeaders": "host",
	}
	canonicalQuery := canonicalQueryString(query)

	canonicalHeaders := "host:" + host + "\n"
	signedHeaders := "host"
	payloadHash := "UNSIGNED-PAYLOAD"

	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	hashedRequest := sha256Hex(canonicalRequest)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hashedRequest,
	}, "\n")

	signingKey := hmacSHA256([]byte("AWS4"+c.SecretAccessKey), dateStamp)
	signingKey = hmacSHA256(signingKey, region)
	signingKey = hmacSHA256(signingKey, service)
	signingKey = hmacSHA256(signingKey, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	query["X-Amz-Signature"] = signature
	encodedQuery := canonicalQueryString(query)

	scheme := "https://"
	if strings.HasPrefix(c.Endpoint, "http://") {
		scheme = "http://"
	}
	return scheme + host + canonicalURI + "?" + encodedQuery, nil
}

func canonicalQueryString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, awsEncode(key)+"="+awsEncode(params[key]))
	}
	return strings.Join(parts, "&")
}

func encodePath(key string) string {
	segments := strings.Split(key, "/")
	for i, segment := range segments {
		segments[i] = awsEncode(segment)
	}
	return strings.Join(segments, "/")
}

func awsEncode(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' || ch == '~' {
			b.WriteByte(ch)
			continue
		}
		b.WriteString("%")
		b.WriteString(strings.ToUpper(hex.EncodeToString([]byte{ch})))
	}
	return b.String()
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	return mac.Sum(nil)
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
