# Go Cloud Utils

A collection of Go utilities for working with popular cloud services.

[![Go Reference](https://pkg.go.dev/badge/github.com/BLANK-13/go-cloud-utils.svg)](https://pkg.go.dev/github.com/BLANK-13/go-cloud-utils)
[![Go Report Card](https://goreportcard.com/badge/github.com/BLANK-13/go-cloud-utils)](https://goreportcard.com/report/github.com/BLANK-13/go-cloud-utils)

## Features

This package provides Go utilities for:

- **Firebase Authentication**: User management, token verification, and middleware for protecting routes
- **Cloudflare Storage**: Interact with Cloudflare D1 (SQL), R2 (Object Storage), and KV (Key-Value Store)

## Installation

```bash
go get github.com/BLANK-13/go-cloud-utils
```

## Firebase Authentication

Firebase Auth utilities provide a clean, reusable interface for authenticating users in Go applications.

### Quick Start

```go
import (
    "context"
    "log"
    "net/http"
    
    "github.com/BLANK-13/go-cloud-utils/firebase"
)

func main() {
    ctx := context.Background()
    
    // Initialize Firebase Auth
    auth, err := firebase.InitFirebase(ctx, "path/to/firebase-service-account.json")
    if err != nil {
        log.Fatalf("Failed to initialize Firebase: %v", err)
    }
    
    // Protect routes with auth middleware
    http.HandleFunc("/public", publicHandler)
    http.HandleFunc("/protected", firebase.RequireAuth(auth, protectedHandler))
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func publicHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Public endpoint")
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    // Get user ID from context (added by auth middleware)
    uid, err := firebase.GetUIDFromContext(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    fmt.Fprintf(w, "Protected endpoint. Hello, %s!", uid)
}
```

### Features

- Token verification and validation
- User management (create, read, update, delete)
- Auth middleware for HTTP handlers
- Custom token creation
- Token revocation

## Cloudflare Storage

Cloudflare storage utilities provide clean interfaces for working with Cloudflare's three storage products.

### Quick Start

```go
import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/BLANK-13/go-cloud-utils/cloudflare"
)

func main() {
    // Initialize Cloudflare storage client
    config := cloudflare.NewConfig(
        os.Getenv("CLOUDFLARE_API_TOKEN"),
        os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
    )
    
    // Create a unified storage client
    storage := cloudflare.NewStorage(config)
    
    // Or only use the specific client you need
    d1Client := cloudflare.NewD1Client(config)
    r2Client := cloudflare.NewR2Client(config)
    kvClient := cloudflare.NewKVClient(config)
    
    ctx := context.Background()
    
    // D1 Database example
    result, err := storage.D1.ExecuteQuery(ctx, "your-database-id", 
        "SELECT * FROM users WHERE name = ?", []interface{}{"John"})
    if err != nil {
        log.Fatalf("Failed to query database: %v", err)
    }
    
    // R2 Object Storage example
    // Upload an image
    file, _ := os.Open("image.jpg")
    defer file.Close()
    obj, err := storage.R2.UploadObject(ctx, "your-bucket", "images/profile.jpg", 
        file, "image/jpeg", map[string]string{"user": "john"})
    if err != nil {
        log.Fatalf("Failed to upload object: %v", err)
    }
    
    // KV example
    // Store a value with 1-hour expiration
    expiration := int64(3600) // 1 hour in seconds
    err = storage.KV.WriteValue(ctx, "your-namespace-id", "greeting", 
        []byte("Hello, World!"), &expiration)
    if err != nil {
        log.Fatalf("Failed to write value: %v", err)
    }
    
    // Read a JSON value
    var user struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    err = storage.KV.ReadJSON(ctx, "your-namespace-id", "user:123", &user)
    if err != nil {
        log.Fatalf("Failed to read JSON: %v", err)
    }
}
```

### Features

#### D1 SQL Database
- Execute SQL queries with parameters
- Process query results

#### R2 Object Storage
- List, upload, download, and delete objects
- Manage object metadata

#### KV Key-Value Store
- List, write, read, and delete values
- Set expiration times for values
- Store and retrieve JSON data

## License

MIT