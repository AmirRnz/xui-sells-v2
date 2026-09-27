package http

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"

	"xui-sells-v2/internal/app/entitlement"
	"xui-sells-v2/internal/domain"
)

var (
	ErrInvalidCredentials = errors.New("invalid telegram ID or password")
	ErrSessionExpired     = errors.New("session expired or invalid")
	ErrUnauthorized       = errors.New("authentication required")
	ErrForbidden          = errors.New("access forbidden")
)

type contextKey string

const sessionContextKey contextKey = "reseller_session"

// Session represents an authenticated reseller session.
type Session struct {
	Token          string
	TelegramID     int64
	InstanceID     int64
	ResellerUserID int64
	Tier           domain.ResellerTier
	ExpiresAt      time.Time
}

// SessionManager manages active in-memory web sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager creates a session store.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession generates a cryptographic session token and stores the session.
func (sm *SessionManager) CreateSession(tgID, instanceID, userID int64, tier domain.ResellerTier, duration time.Duration) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	s := &Session{
		Token:          token,
		TelegramID:     tgID,
		InstanceID:     instanceID,
		ResellerUserID: userID,
		Tier:           tier,
		ExpiresAt:      time.Now().Add(duration),
	}

	sm.sessions[token] = s
	return s, nil
}

// GetSession validates and retrieves a session.
func (sm *SessionManager) GetSession(token string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[token]
	if !ok {
		return nil, ErrSessionExpired
	}
	if time.Now().After(s.ExpiresAt) {
		return nil, ErrSessionExpired
	}
	return s, nil
}

// DeleteSession destroys a session.
func (sm *SessionManager) DeleteSession(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, token)
}

// HashPassword hashes a plain-text password using Argon2id.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Standard recommended Argon2id parameters
	timeCost := uint32(1)
	memoryCost := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)

	hash := argon2.IDKey([]byte(password), salt, timeCost, memoryCost, threads, keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryCost, timeCost, threads, b64Salt, b64Hash)

	return encoded, nil
}

// VerifyPassword verifies a password against an Argon2id formatted hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid argon2id hash format")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, err
	}

	var memoryCost, timeCost uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memoryCost, &timeCost, &threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	hash := argon2.IDKey([]byte(password), salt, timeCost, memoryCost, threads, uint32(len(expectedHash)))

	if subtle.ConstantTimeCompare(hash, expectedHash) == 1 {
		return true, nil
	}
	return false, nil
}

// AuthMiddleware inspects bearer tokens or session cookies and injects the session into request context.
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""

		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if cookie, err := r.Cookie("reseller_token"); err == nil {
			token = cookie.Value
		}

		if token == "" {
			http.Error(w, `{"error":"unauthorized","message":"Authorization token required"}`, http.StatusUnauthorized)
			return
		}

		sess, err := s.sessions.GetSession(token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized","message":"Session invalid or expired"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireWebPanelMiddleware checks that the reseller tier has permission to access the web panel (D07 & P17).
func (s *Server) RequireWebPanelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GetSessionFromContext(r.Context())
		if sess == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		resolver := entitlement.NewResolver()
		if !resolver.CanAccessWebPanel(sess.Tier) {
			http.Error(w, `{"error":"forbidden","message":"Web panel access requires an active Ultimate tier membership"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetSessionFromContext extracts the session from request context.
func GetSessionFromContext(ctx context.Context) *Session {
	val := ctx.Value(sessionContextKey)
	if val == nil {
		return nil
	}
	sess, ok := val.(*Session)
	if !ok {
		return nil
	}
	return sess
}
