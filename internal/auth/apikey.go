package auth

// APIKeyAuth provides a simple API key authentication
type APIKeyAuth struct {
	validKeys map[string]string
}

// NewAPIKeyAuth creates a new API key authentication middleware
func NewAPIKeyAuth(keys []string) *APIKeyAuth {
	validKeys := make(map[string]string)
	for _, key := range keys {
		validKeys[key] = "valid"
	}

	return &APIKeyAuth{
		validKeys: validKeys,
	}
}

// AddKey adds a new valid API key
func (a *APIKeyAuth) AddKey(key string) {
	a.validKeys[key] = "valid"
}

// RemoveKey removes a valid API key
func (a *APIKeyAuth) RemoveKey(key string) {
	delete(a.validKeys, key)
}

// IsValidKey checks if a key is valid
func (a *APIKeyAuth) IsValidKey(key string) bool {
	_, valid := a.validKeys[key]
	return valid
}
