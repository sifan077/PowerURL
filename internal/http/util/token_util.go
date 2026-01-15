package util

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken  = errors.New("invalid or expired token")
	ErrMissingSecret = errors.New("redirect secret is not configured")
)

// TokenSigner encapsulates HMAC issuance/validation so handlers stay small.
type TokenSigner struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenSigner returns a signer that issues compact HMAC tokens.
func NewTokenSigner(secret []byte, ttl time.Duration) *TokenSigner {
	return &TokenSigner{
		secret: secret,
		ttl:    ttl,
	}
}

// Issue mints a token for the provided short-code.
func (s *TokenSigner) Issue(code string) (string, error) {
	return s.IssueWithClickID(code, "")
}

// IssueWithClickID mints a token for the provided short-code with optional click event ID.
func (s *TokenSigner) IssueWithClickID(code, clickID string) (string, error) {
	if len(s.secret) == 0 {
		return "", ErrMissingSecret
	}

	// Base payload: 4 bytes expiry + 8 random bytes
	basePayload := make([]byte, 12)
	expires := uint32(time.Now().Add(s.ttl).Unix())
	binary.BigEndian.PutUint32(basePayload[:4], expires)
	if _, err := rand.Read(basePayload[4:]); err != nil {
		return "", err
	}

	// Combine base payload with click ID
	var payload []byte
	if clickID != "" {
		// Format: clickID length (2 bytes) + clickID + basePayload
		clickIDBytes := []byte(clickID)
		clickIDLen := uint16(len(clickIDBytes))
		payload = make([]byte, 2+len(clickIDBytes)+len(basePayload))
		binary.BigEndian.PutUint16(payload[:2], clickIDLen)
		copy(payload[2:2+len(clickIDBytes)], clickIDBytes)
		copy(payload[2+len(clickIDBytes):], basePayload)
	} else {
		payload = basePayload
	}

	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(code, payload)
	sigEnc := base64.RawURLEncoding.EncodeToString(signature[:16])
	return fmt.Sprintf("%s.%s", payloadEnc, sigEnc), nil
}

// Validate checks signature integrity and TTL of the token.
func (s *TokenSigner) Validate(code, token string) error {
	_, err := s.ValidateAndExtractClickID(code, token)
	return err
}

// ValidateAndExtractClickID checks signature integrity and TTL of the token, and extracts the click event ID if present.
func (s *TokenSigner) ValidateAndExtractClickID(code, token string) (string, error) {
	if len(s.secret) == 0 {
		return "", ErrMissingSecret
	}

	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidToken
	}

	sigProvided, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalidToken
	}
	if len(sigProvided) != 16 {
		return "", ErrInvalidToken
	}

	expected := s.sign(code, payload)
	if !hmac.Equal(sigProvided, expected[:16]) {
		return "", ErrInvalidToken
	}

	if len(payload) < 4 {
		return "", ErrInvalidToken
	}

	// Check if payload contains click ID
	var clickID string
	var basePayload []byte
	if len(payload) > 14 { // 2 bytes length + at least 1 byte clickID + 12 bytes basePayload
		clickIDLen := binary.BigEndian.Uint16(payload[:2])
		if int(clickIDLen)+2+12 <= len(payload) {
			clickID = string(payload[2 : 2+clickIDLen])
			basePayload = payload[2+clickIDLen:]
		} else {
			basePayload = payload
		}
	} else {
		basePayload = payload
	}

	expires := binary.BigEndian.Uint32(basePayload[:4])
	if time.Now().Unix() > int64(expires) {
		return "", ErrInvalidToken
	}

	return clickID, nil
}

func (s *TokenSigner) sign(code string, payload []byte) []byte {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(code))
	mac.Write([]byte("|"))
	mac.Write(payload)
	return mac.Sum(nil)
}
