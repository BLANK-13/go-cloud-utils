package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	officialFirebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/BLANK-13/go-cloud-utils/cloudflare"
	"github.com/BLANK-13/go-cloud-utils/firebase"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	auth, err := firebase.InitFirebase(ctx, os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON_PATH"))
	
	if err != nil {
		log.Printf("Failed to initialize Firebase: %v", err)
		return
	}

	// Example: Firebase Authentication
	firebaseExample(ctx, auth)

	// Example: BaseUser model
	userModelExample(ctx, auth)

	// Example: Cloudflare Storage
	cloudflareExample(ctx)

	//Example: a simple protected endpoint with the firebase util
	startWebServer(auth)
}

func firebaseExample(ctx context.Context, auth *firebase.FirebaseAuth) {
	fmt.Println("=== Firebase Authentication Example ===")

	// Initialize Firebase with service account
	// In production, you should use environment variables

	// Example: Create a user
	userParams := (&officialFirebaseAuth.UserToCreate{}).
		Email("test@example.com").
		Password("password123").
		DisplayName("Test User")

	uid, err := auth.CreateUser(ctx, userParams)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
	} else {
		log.Printf("Created user with UID: %s", uid)

		// Example: Set custom claims
		claims := map[string]interface{}{
			"admin": true,
			"level": 5,
		}

		if err := auth.SetCustomClaims(ctx, uid, claims); err != nil {
			log.Printf("Failed to set custom claims: %v", err)
		} else {
			log.Printf("Set custom claims for user: %s", uid)
		}

		// Example: Get user details
		user, err := auth.GetUserByUID(ctx, uid)
		if err != nil {
			log.Printf("Failed to get user: %v", err)
		} else {
			log.Printf("User details: %s (%s), %v", user.DisplayName, user.Email, user.CustomClaims)
		}

		// Clean up - delete user
		if err := auth.DeleteUser(ctx, uid); err != nil {
			log.Printf("Failed to delete user: %v", err)
		} else {
			log.Printf("Deleted user: %s", uid)
		}
	}
}

func userModelExample(ctx context.Context, auth *firebase.FirebaseAuth) {
	fmt.Println("\n=== BaseUser Model Example ===")

	// Define an application-specific user data type
	type AppUserData struct {
		Preferences     map[string]string `json:"preferences"`
		LastActive      time.Time         `json:"lastActive"`
		ProfileComplete bool              `json:"profileComplete"`
	}

	// Create a test user
	userParams := (&officialFirebaseAuth.UserToCreate{}).
		Email("baseuser@example.com").
		Password("password123").
		DisplayName("Base User Test")

	uid, err := auth.CreateUser(ctx, userParams)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return
	}
	log.Printf("Created user with Firebase UID: %s", uid)

	// Set some custom claims
	claims := map[string]interface{}{
		"admin": true,
		"roles": []interface{}{"editor", "viewer"},
		"level": 5,
	}

	if err := auth.SetCustomClaims(ctx, uid, claims); err != nil {
		log.Printf("Failed to set custom claims: %v", err)
	}

	// Get the Firebase user
	fbUser, err := auth.GetUserByUID(ctx, uid)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}

	// Create app-specific user data
	userData := AppUserData{
		Preferences: map[string]string{
			"theme":    "dark",
			"language": "en",
		},
		LastActive:      time.Now(),
		ProfileComplete: false,
	}

	// Convert to our BaseUser model
	baseUser := firebase.FromFirebaseUser[AppUserData](fbUser, userData)

	// Display the user model
	log.Printf("BaseUser created with ID: %s (different from Firebase UID: %s)",
		baseUser.ID, baseUser.FirebaseUID)

	// Demonstrate claims/roles helpers
	isAdmin, _ := baseUser.GetClaim("admin")
	log.Printf("User is admin: %v", isAdmin)

	log.Printf("User has editor role: %v", baseUser.HasRole("editor"))
	log.Printf("User has admin role: %v", baseUser.HasRole("admin"))

	// Show app-specific data
	log.Printf("User preferences: %v", baseUser.Data.Preferences)
	log.Printf("Profile complete: %v", baseUser.Data.ProfileComplete)

	// Clean up - delete user
	if err := auth.DeleteUser(ctx, uid); err != nil {
		log.Printf("Failed to delete user: %v", err)
	} else {
		log.Printf("Deleted user: %s", uid)
	}
}

func cloudflareExample(ctx context.Context) {
	fmt.Println("\n=== Cloudflare Storage Example ===")

	// Get credentials from environment
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	if apiToken == "" || accountID == "" {
		log.Println("CLOUDFLARE_API_TOKEN or CLOUDFLARE_ACCOUNT_ID environment variables not set")
		return
	}

	// Create configuration
	config := cloudflare.NewConfig(apiToken, accountID)

	// Initialize all storage clients
	storage := cloudflare.NewStorage(config)

	// Example: Database ID, Bucket Name, and Namespace ID would typically come from environment variables
	databaseID := os.Getenv("CLOUDFLARE_D1_DATABASE_ID")
	bucketName := os.Getenv("CLOUDFLARE_R2_BUCKET_NAME")
	namespaceID := os.Getenv("CLOUDFLARE_KV_NAMESPACE_ID")

	// Example: D1 Database operations
	// NOTE: let's assume your database has a user table with the following schema for the sake of this example [id:int, name:text, email:text]

	if databaseID != "" {

		user := "John" + time.Now().Format("2006-01-02 15:04:05")
		email := "john@example.com"

		// 1. Insert a user
		log.Println("Inserting user...")
		_, err := storage.D1.ExecuteQuery(ctx, databaseID,
			"INSERT INTO users (name, email) VALUES (?, ?)",
			[]interface{}{user, email})
		if err != nil {
			log.Printf("Failed to insert user: %v", err)
		} else {
			log.Println("User inserted successfully")

			// 2. Query the user (existing code)
			log.Println("Executing D1 database query...")
			result, err := storage.D1.ExecuteQuery(ctx, databaseID,
				"SELECT * FROM users WHERE name = ?",
				[]interface{}{"user"})
			if err != nil {
				log.Printf("D1 query failed: %v", err)
			} else {
				log.Printf("D1 query successful: %d rows returned", len(result.Results))
				// log.Printf("D1 query results: %v", result.Results)
			}
		}

	}

	// Example: R2 Object Storage operations
	if bucketName != "" {
		log.Println("Listing R2 objects...")
		objects, err := storage.R2.ListObjects(ctx, bucketName, "")
		if err != nil {
			log.Printf("R2 list objects failed: %v", err)
		} else {
			log.Printf("R2 list objects successful: %d objects found", len(objects))
		}

		// Example: Upload object
		log.Println("Uploading object to R2...")
		data := []byte("Hello, world!")
		obj, err := storage.R2.UploadObject(ctx, bucketName, "test.txt",
			bytes.NewReader(data), "text/plain",
			map[string]string{"source": "example"})
		if err != nil {
			log.Printf("R2 upload failed: %v", err)
		} else {
			log.Printf("R2 upload successful: %s", obj.Key)
		}
	}

	// Example: KV operations
	if namespaceID != "" {
		log.Println("Writing to KV...")
		value := []byte("Hello, world!")
		expiration := int64(3600) // 1 hour in seconds

		err := storage.KV.WriteValue(ctx, namespaceID, "test-key", value, &expiration)
		if err != nil {
			log.Printf("KV write failed: %v", err)
		} else {
			log.Println("KV write successful")

			// Read back the value
			readValue, err := storage.KV.ReadValue(ctx, namespaceID, "test-key")
			if err != nil {
				log.Printf("KV read failed: %v", err)
			} else {
				log.Printf("KV read successful: %s", string(readValue))
			}
		}
	}
}

// startWebServer sets up routes and starts the HTTP server
func startWebServer(auth *firebase.FirebaseAuth) {
	fmt.Println("\n=== Starting Web Server ===")

	http.HandleFunc("/public", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Public endpoint")
	})

	// Use the RequireAuth middleware from your package
	http.HandleFunc("/protected", firebase.RequireAuth(auth, func(w http.ResponseWriter, r *http.Request) {
		uid, err := firebase.GetUIDFromContext(r.Context())
		if err != nil {
			// This should ideally not happen if RequireAuth worked, but good practice
			http.Error(w, "Failed to get UID from context", http.StatusInternalServerError)
			return
		}
		// You could also get the full token if needed:
		// token, _ := firebase.GetTokenFromContext(r.Context())
		// fmt.Fprintf(w, "Protected endpoint. Hello, %s! Your claims: %v", uid, token.Claims)

		fmt.Fprintf(w, "Protected endpoint. Hello, %s!", uid)
	}))

	log.Println("Registered HTTP handlers:")
	log.Println("  /public")
	log.Println("  /protected (requires Bearer token)")
	log.Println("Starting server on http://localhost:8080 ...")

	// http.ListenAndServe will block execution unless there's an error
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
