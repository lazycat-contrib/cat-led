package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const sessionCookie = "cat_led_session"

// A process-local key deliberately expires sessions on restart. Identity and
// role cookies from older releases are never accepted as authenticated input.
var sessionKey = func() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}()

type sessionIdentity struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Expires int64  `json:"expires"`
}

func encodeSession(id, role string) string {
	payload, _ := json.Marshal(sessionIdentity{id, role, time.Now().Add(24 * time.Hour).Unix()})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, sessionKey)
	mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func decodeSession(value string) (sessionIdentity, bool) {
	var identity sessionIdentity
	payload, signature, ok := strings.Cut(value, ".")
	if !ok {
		return identity, false
	}
	sig, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return identity, false
	}
	mac := hmac.New(sha256.New, sessionKey)
	mac.Write([]byte(payload))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return identity, false
	}
	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil || json.Unmarshal(data, &identity) != nil {
		return identity, false
	}
	return identity, identity.ID != "" && identity.Expires > time.Now().Unix()
}

// UserID accepts the gateway-injected identity or a verified OIDC session.
// The application port must only be exposed through the LazyCat gateway.
func UserID(c *gin.Context) string {
	if id := c.GetHeader("x-hc-user-id"); id != "" {
		return id
	}
	return c.GetString("user_id")
}
