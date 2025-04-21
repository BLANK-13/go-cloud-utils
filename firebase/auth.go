package firebase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// FirebaseAuth holds the Firebase client and auth client instances
type FirebaseAuth struct {
	App  *firebase.App
	Auth *auth.Client
}

// InitFirebase initializes Firebase with the provided service account key file
// Example usage:
//
//	ctx := context.Background()
//	firebaseAuth, err := InitFirebase(ctx, "path/to/service-account.json")
//	if err != nil {
//	    log.Fatalf("Failed to initialize Firebase: %v", err)
//	}
func InitFirebase(ctx context.Context, serviceAccountKeyPath string) (*FirebaseAuth, error) {
	opt := option.WithCredentialsFile(serviceAccountKeyPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase: %v", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Auth client: %v", err)
	}

	return &FirebaseAuth{
		App:  app,
		Auth: client,
	}, nil
}

// InitFirebaseWithCredentials initializes Firebase with a Google credentials JSON string
func InitFirebaseWithCredentials(ctx context.Context, credentialsJSON []byte) (*FirebaseAuth, error) {
	opt := option.WithCredentialsJSON(credentialsJSON)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase: %v", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting Auth client: %v", err)
	}

	return &FirebaseAuth{
		App:  app,
		Auth: client,
	}, nil
}

// VerifyIDToken verifies the ID token and returns the Firebase token
// Example usage:
//
//	token, err := firebaseAuth.VerifyIDToken(ctx, idToken)
//	if err != nil {
//	    http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	    return
//	}
//	// Use token.UID to identify the user
func (fa *FirebaseAuth) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	if idToken == "" {
		return nil, errors.New("id token is empty")
	}

	token, err := fa.Auth.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("error verifying ID token: %v", err)
	}

	return token, nil
}

// GetUserByUID gets a user by their Firebase UID
func (fa *FirebaseAuth) GetUserByUID(ctx context.Context, uid string) (*auth.UserRecord, error) {
	user, err := fa.Auth.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %v", err)
	}

	return user, nil
}

// GetUserByEmail gets a user by their email address
func (fa *FirebaseAuth) GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	user, err := fa.Auth.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("error getting user by email: %v", err)
	}

	return user, nil
}

// CreateUser creates a new Firebase user
func (fa *FirebaseAuth) CreateUser(ctx context.Context, params *auth.UserToCreate) (string, error) {
	user, err := fa.Auth.CreateUser(ctx, params)
	if err != nil {
		return "", fmt.Errorf("error creating user: %v", err)
	}

	return user.UID, nil
}

// UpdateUser updates an existing Firebase user
func (fa *FirebaseAuth) UpdateUser(ctx context.Context, uid string, params *auth.UserToUpdate) error {
	_, err := fa.Auth.UpdateUser(ctx, uid, params)
	if err != nil {
		return fmt.Errorf("error updating user: %v", err)
	}

	return nil
}

// DeleteUser deletes a Firebase user
func (fa *FirebaseAuth) DeleteUser(ctx context.Context, uid string) error {
	err := fa.Auth.DeleteUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("error deleting user: %v", err)
	}

	return nil
}

// CreateCustomToken creates a custom token for a user
func (fa *FirebaseAuth) CreateCustomToken(ctx context.Context, uid string, claims map[string]interface{}) (string, error) {
	var token string
	var err error

	if claims != nil && len(claims) > 0 {
		token, err = fa.Auth.CustomTokenWithClaims(ctx, uid, claims)
	} else {
		token, err = fa.Auth.CustomToken(ctx, uid)
	}

	if err != nil {
		return "", fmt.Errorf("error creating custom token: %v", err)
	}

	return token, nil
}

// RevokeTokens revokes all refresh tokens for a user
func (fa *FirebaseAuth) RevokeTokens(ctx context.Context, uid string) error {
	err := fa.Auth.RevokeRefreshTokens(ctx, uid)
	if err != nil {
		return fmt.Errorf("error revoking tokens: %v", err)
	}

	return nil
}

// GetTokenFromRequest extracts the ID token from the Authorization header
func GetTokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// Check if the header starts with "Bearer "
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer "), nil
	}

	return "", errors.New("authorization header must be in the format 'Bearer {token}'")
}

// AuthMiddleware is a middleware to verify Firebase authentication tokens
// Example usage with standard http:
//
//	http.Handle("/protected", firebaseAuth.AuthMiddleware(protectedHandler))
//
// Example with Gorilla mux:
//
//	router.Handle("/protected", firebaseAuth.AuthMiddleware(protectedHandler))
func (fa *FirebaseAuth) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idToken, err := GetTokenFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Verify the token
		token, err := fa.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Add the token to the request context
		ctx := context.WithValue(r.Context(), "token", token)
		ctx = context.WithValue(ctx, "uid", token.UID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth is a utility function to be used with http handlers as middleware
// Example usage:
//
//	http.HandleFunc("/protected", RequireAuth(firebaseAuth, protectedHandlerFunc))
func RequireAuth(fa *FirebaseAuth, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idToken, err := GetTokenFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Verify the token
		token, err := fa.VerifyIDToken(r.Context(), idToken)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Add the token to the request context
		ctx := context.WithValue(r.Context(), "token", token)
		ctx = context.WithValue(ctx, "uid", token.UID)

		// Call the next handler with the updated context
		next(w, r.WithContext(ctx))
	}
}

// GetUIDFromContext extracts the user ID from the context
func GetUIDFromContext(ctx context.Context) (string, error) {
	uid, ok := ctx.Value("uid").(string)
	if !ok {
		return "", errors.New("unauthorized: uid not found in context")
	}
	return uid, nil
}

// GetTokenFromContext extracts the token from the context
func GetTokenFromContext(ctx context.Context) (*auth.Token, error) {
	token, ok := ctx.Value("token").(*auth.Token)
	if !ok {
		return nil, errors.New("unauthorized: token not found in context")
	}
	return token, nil
}
