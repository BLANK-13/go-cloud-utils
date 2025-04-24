package firebase

import (
	"time"

	"firebase.google.com/go/v4/auth"
)

// BaseUser contains universal user properties across any application
type BaseUser[T any] struct {
	// Core identifiers
	ID          string `json:"id"`    // Database primary key/ID
	Email       string `json:"email"` // User's email address
	PhoneNumber string `json:"phoneNumber,omitempty"`

	// Common user metadata
	DisplayName string     `json:"displayName,omitempty"`
	PhotoURL    string     `json:"photoUrl,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	Disabled    bool       `json:"disabled"`

	// Authentication properties
	EmailVerified bool                   `json:"emailVerified"`
	PhoneVerified bool                   `json:"phoneVerified"`
	Claims        map[string]interface{} `json:"claims,omitempty"`

	// Application-specific user data
	Data T `json:"data"`
}

// FromFirebaseUser converts a Firebase UserRecord to a BaseUser
func FromFirebaseUser[T any](user *auth.UserRecord, data T) *BaseUser[T] {
	lastLogin := user.UserMetadata.LastLogInTimestamp
	var lastLoginTime *time.Time

	if lastLogin > 0 {
		t := time.Unix(lastLogin/1000, 0)
		lastLoginTime = &t
	}

	createdAt := time.Unix(user.UserMetadata.CreationTimestamp/1000, 0)

	// New fields for phone authentication
	phoneVerified := user.PhoneNumber != ""

	return &BaseUser[T]{
		ID:            user.UID,
		Email:         user.Email,
		DisplayName:   user.DisplayName,
		PhotoURL:      user.PhotoURL,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
		LastLoginAt:   lastLoginTime,
		Disabled:      user.Disabled,
		EmailVerified: user.EmailVerified,
		Claims:        user.CustomClaims,
		Data:          data,
		PhoneNumber:   user.PhoneNumber,
		PhoneVerified: phoneVerified,
	}
}

// ToFirebaseUpdate converts a BaseUser to auth.UserToUpdate for Firebase updates
func (u *BaseUser[T]) ToFirebaseUpdate() *auth.UserToUpdate {
	update := (&auth.UserToUpdate{}).
		Email(u.Email).
		DisplayName(u.DisplayName).
		PhotoURL(u.PhotoURL).
		EmailVerified(u.EmailVerified).
		Disabled(u.Disabled)

	return update
}

// GetClaim retrieves a specific claim value with type safety
func (u *BaseUser[T]) GetClaim(key string) (interface{}, bool) {
	if u.Claims == nil {
		return nil, false
	}

	val, ok := u.Claims[key]
	return val, ok
}

// HasRole checks if a user has a specific role
func (u *BaseUser[T]) HasRole(role string) bool {
	if u.Claims == nil {
		return false
	}

	// Check roles array if it exists
	if roles, ok := u.Claims["roles"]; ok {
		if rolesArr, ok := roles.([]interface{}); ok {
			for _, r := range rolesArr {
				if r == role {
					return true
				}
			}
		}
	}

	// Check direct role claim
	if val, ok := u.Claims[role]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}

	return false
}
