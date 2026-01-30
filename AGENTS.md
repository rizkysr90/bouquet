# 🌸 Flower Supply Catalog - Project Context (Simplified)

**Version:** 1.0  
**Developer:** Rizki  
**Tool:** Cursor AI  
**Timeline:** 2-3 weeks

---

## 1. Executive Summary

### Project Overview
Build a **product catalog website** for a flower supply retail business with:
- **Public catalog** - Customers browse products, see variants, prices
- **Admin backoffice** - Owner manages products, uploads photos, updates info
- **WhatsApp integration** - Customers contact seller via WA (no online payment/cart)

### Tech Stack
- **Backend:** Go (Fiber framework)
- **Frontend:** htmx + Tailwind CSS
- **Database:** PostgreSQL
- **Image Storage:** Cloudinary
- **Deployment:** Railway (Docker)
- **Development:** Docker Compose

### Key Features
**Public:**
- Browse products (grid view)
- Search by name/code
- Filter by category, price, availability
- View product details + variants
- Contact seller via WhatsApp

**Admin:**
- Login/logout (simple JWT auth)
- CRUD products
- Upload photos (main + variants)
- Manage categories
- Set sale/soldout flags

### Success Metrics
- Page load < 2s
- Admin can add product in < 2 minutes
- Mobile-friendly (80% traffic mobile)
- Cost < $10/month
- 99% uptime

---

## 2. Business Context

### The Business
- **Industry:** Retail bahan baku buket bunga
- **Products:** Wrapping paper, ribbons, accessories (~1000 SKUs)
- **Customers:** Florists, DIY crafters, event organizers
- **Current State:** Manual sharing via WhatsApp/Instagram (inefficient)

### Business Model
**This is NOT an e-commerce site**

```
Customer Journey:
1. Browse catalog on website
2. Find products they want
3. Click "Contact Seller" (WhatsApp)
4. Chat with owner on WhatsApp
5. Negotiate, confirm order
6. Payment via bank transfer
7. Pickup/delivery arranged privately
```

**Why No Online Payment?**
- Client prefers personal interaction
- Flexible pricing (bulk discounts, negotiation)
- Simpler implementation (MVP)
- Lower risk (no payment gateway complexity)

### User Personas

**Customer (Public User):**
- Mobile-first (smartphones)
- Quick browsing (visual learners)
- Needs: See all products, check availability, get prices
- Action: Browse → Find → WhatsApp seller

**Admin (Store Owner):**
- Desktop user (laptop/tablet)
- Not tech-savvy (needs simple UI)
- Needs: Add/edit products easily, upload photos
- Action: Login → Manage products → Logout

### Business Goals
1. **Reduce manual inquiries** - Self-service catalog
2. **Professional online presence** - Build brand trust
3. **Showcase full inventory** - All 1000 products visible
4. **Easy updates** - Owner can update prices/stock independently
5. **Foundation for growth** - Can add e-commerce later

---

## 3. Technical Requirements

### Functional Requirements

#### Public Catalog (FR-PUB)

**FR-PUB-001: Product Listing**
- Display products in grid (responsive)
- Show: photo, title, price, sale/soldout badge
- Pagination: 20 items per page

**FR-PUB-002: Search**
- Search by: product name, product code
- Real-time search (debounced 500ms)

**FR-PUB-003: Filter & Sort**
- Filter by: Category, Price range, Availability
- Sort by: Newest, Price (low-high, high-low), Name (A-Z)

**FR-PUB-004: Product Detail**
- View: All photos, description, variants, prices
- Select variant → Update photo & price
- CTA: "Chat di WhatsApp" button

**FR-PUB-005: WhatsApp Integration**
- Button opens WhatsApp with pre-filled message:
  ```
  "Halo, saya tertarik dengan [Product Title] - [Variant Color]. Apakah masih tersedia?"
  ```
- Seller's WhatsApp number configured in env var

#### Admin Backoffice (FR-ADM)

**FR-ADM-001: Authentication**
- Login page (username + password)
- JWT token (24h expiry, HttpOnly cookie)
- Logout (clear session)

**FR-ADM-002: Product Management**
- List products (table view, search, pagination)
- Create product (form with validation)
- Edit product (pre-filled form)
- Delete product (confirmation modal)

**FR-ADM-003: Image Upload**
- Upload to Cloudinary
- Supported: JPG, PNG, WebP (max 5MB)
- Auto-optimize (quality, format)
- Preview before save

**FR-ADM-004: Variant Management**
- Add/edit/delete color variants
- Each variant: color name, photo, price adjustment
- Actual price = base_price + price_adjustment

**FR-ADM-005: Category Management**
- List categories
- Create/edit/delete category
- Auto-generate slug from name

### Non-Functional Requirements

**NFR-001: Performance**
- Page load: < 2s (public catalog)
- API response: < 200ms (average)
- Database query: < 50ms (with indexes)

**NFR-002: Scalability**
- Support 1000-5000 products
- Handle 100 concurrent users
- Stateless app (horizontal scaling ready)

**NFR-003: Security**
- HTTPS only (production)
- Password hashing (bcrypt cost 10)
- JWT secret rotation capability
- SQL injection prevention (parameterized queries)
- XSS prevention (template auto-escaping)

**NFR-004: Reliability**
- 99.5% uptime target
- Graceful error handling
- Database connection retry
- Health check endpoint

**NFR-005: Maintainability**
- Clean architecture (handlers → services → repos)
- Environment-based configuration
- Docker for reproducible environments
- Migration files for database schema

**NFR-006: Usability**
- Mobile-first responsive design
- Accessible (keyboard navigation, alt text)
- Loading indicators for async operations
- Clear error messages

---

## 4. System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────┐
│             User Browser                    │
│   (Public: Mobile/Desktop)                  │
│   (Admin: Desktop)                          │
└──────────────┬──────────────────────────────┘
               │ HTTPS
               │
┌──────────────▼──────────────────────────────┐
│          Railway (Production)               │
│  ┌────────────────────────────────────────┐ │
│  │   Go Application (Docker Container)    │ │
│  │   - Fiber Web Server                   │ │
│  │   - htmx + Tailwind (rendered HTML)    │ │
│  │   - Port: 3000                         │ │
│  └────┬───────────────────────┬───────────┘ │
│       │                       │              │
│  ┌────▼──────────┐      ┌────▼──────────┐  │
│  │  PostgreSQL   │      │  Health Check │  │
│  │  (Railway     │      │  /health      │  │
│  │   Plugin)     │      └───────────────┘  │
│  └───────────────┘                          │
└─────────────────────────────────────────────┘
               │
               │ API Calls
               │
┌──────────────▼──────────────────────────────┐
│          Cloudinary CDN                     │
│  - Image Storage                            │
│  - Auto Optimization (WebP, quality)        │
│  - On-the-fly Transformations               │
└─────────────────────────────────────────────┘
```

### Application Architecture (Layered)

```
┌─────────────────────────────────────────┐
│      Presentation Layer                 │
│  - htmx (interactivity)                 │
│  - Tailwind CSS (styling)               │
│  - HTML Templates (Go html/template)    │
└───────────────┬─────────────────────────┘
                │ HTTP Requests/Responses
┌───────────────▼─────────────────────────┐
│      HTTP Handler Layer                 │
│  - PublicHandler (catalog routes)       │
│  - AdminHandler (CRUD routes)           │
│  - AuthHandler (login/logout)           │
│  - UploadHandler (image upload)         │
└───────────────┬─────────────────────────┘
                │ Call Business Logic
┌───────────────▼─────────────────────────┐
│      Service Layer (Business Logic)     │
│  - ProductService                       │
│  - CategoryService                      │
│  - AuthService                          │
│  - CloudinaryService                    │
└───────────────┬─────────────────────────┘
                │ Data Operations
┌───────────────▼─────────────────────────┐
│      Repository Layer (Data Access)     │
│  - ProductRepository                    │
│  - CategoryRepository                   │
│  - AdminRepository                      │
└───────────────┬─────────────────────────┘
                │ SQL Queries
┌───────────────▼─────────────────────────┐
│      PostgreSQL Database                │
│  - products                             │
│  - product_variants                     │
│  - categories                           │
│  - admins                               │
└─────────────────────────────────────────┘
```

### Component Responsibilities

**Handlers (HTTP Layer):**
- Receive HTTP requests
- Validate input (basic)
- Call service methods
- Return HTTP responses (HTML or JSON)
- Handle errors (convert to user-friendly messages)

**Services (Business Logic):**
- Implement business rules
- Orchestrate operations (e.g., create product + upload image)
- Validate data (comprehensive)
- Coordinate between repositories
- Error handling & logging

**Repositories (Data Access):**
- Execute SQL queries
- Map database rows to structs
- Return errors (no business logic)
- Use parameterized queries (SQL injection prevention)

**Models:**
- Data structures (structs)
- Represent database entities
- No business logic (just data)

### Tech Stack Justification

| Component | Choice | Why? |
|-----------|--------|------|
| **Language** | Go 1.22+ | Performance, type safety, single binary, your expertise |
| **Web Framework** | Fiber v2 | Fast, Express-like, good middleware, easy to learn |
| **Frontend** | htmx + Tailwind | No build step, server-driven, simple, fast development |
| **Database** | PostgreSQL 15+ | Reliable, JSON support, Railway native |
| **Image Storage** | Cloudinary | Free tier (25GB), auto-optimization, CDN, transformations |
| **Containerization** | Docker | Reproducible environments, Railway compatible |
| **Hosting** | Railway | Simple deployment, PostgreSQL included, affordable ($5-10/month) |
| **Auth** | JWT + bcrypt | Stateless, simple, secure enough for single admin |

**Why NOT React/Vue/Next.js?**
- Overkill for CRUD + catalog
- htmx provides sufficient interactivity
- Server-side rendering better for SEO
- No build complexity
- Faster development for solo dev

---

## 5. Data Architecture

### Entity Relationship Diagram

```
┌─────────────────┐
│   categories    │
│─────────────────│
│ id (PK)         │
│ name            │
│ slug (UNIQUE)   │
│ created_at      │
│ updated_at      │
└────────┬────────┘
         │ 1
         │ has many
         │ N
┌────────▼────────────────┐
│      products           │
│─────────────────────────│
│ id (PK)                 │
│ code (UNIQUE)           │───────┐
│ title                   │       │
│ description             │       │
│ main_photo_url          │       │
│ main_photo_id           │       │
│ category_id (FK)        │       │
│ base_price              │       │
│ is_sale                 │       │
│ is_soldout              │       │
│ created_at              │       │
│ updated_at              │       │
└────────┬────────────────┘       │
         │ 1                       │
         │ has many                │
         │ N                       │
┌────────▼──────────────────┐     │
│   product_variants        │     │
│───────────────────────────│     │
│ id (PK)                   │     │
│ product_id (FK) ──────────┼─────┘
│ color                     │
│ photo_url                 │
│ photo_id                  │
│ price_adjustment          │
│ created_at                │
│ updated_at                │
│ UNIQUE(product_id, color) │
└───────────────────────────┘

┌─────────────────┐
│     admins      │
│─────────────────│
│ id (PK)         │
│ username (UNIQUE│
│ password_hash   │
│ created_at      │
└─────────────────┘
```

### Database Schema

#### categories
```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_categories_slug ON categories(slug);
```

#### products
```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    main_photo_url VARCHAR(500),
    main_photo_id VARCHAR(200),      -- Cloudinary public_id (for deletion)
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    base_price DECIMAL(12,2) NOT NULL CHECK (base_price >= 0),
    is_sale BOOLEAN DEFAULT FALSE,
    is_soldout BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_code ON products(code);
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_flags ON products(is_sale, is_soldout);
CREATE INDEX idx_products_search ON products USING gin(to_tsvector('indonesian', title));
```

#### product_variants
```sql
CREATE TABLE product_variants (
    id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    color VARCHAR(50) NOT NULL,
    photo_url VARCHAR(500),
    photo_id VARCHAR(200),           -- Cloudinary public_id
    price_adjustment DECIMAL(12,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, color)
);

CREATE INDEX idx_variants_product ON product_variants(product_id);
```

#### admins
```sql
CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admins_username ON admins(username);
```

### Data Models (Go Structs)

```go
// Product represents a product entity
type Product struct {
    ID            int                `db:"id" json:"id"`
    Code          string             `db:"code" json:"code"`
    Title         string             `db:"title" json:"title"`
    Description   string             `db:"description" json:"description"`
    MainPhotoURL  string             `db:"main_photo_url" json:"main_photo_url"`
    MainPhotoID   string             `db:"main_photo_id" json:"main_photo_id"`
    CategoryID    *int               `db:"category_id" json:"category_id"`
    BasePrice     float64            `db:"base_price" json:"base_price"`
    IsSale        bool               `db:"is_sale" json:"is_sale"`
    IsSoldout     bool               `db:"is_soldout" json:"is_soldout"`
    CreatedAt     time.Time          `db:"created_at" json:"created_at"`
    UpdatedAt     time.Time          `db:"updated_at" json:"updated_at"`
    
    // Relations (not in DB)
    Category      *Category          `db:"-" json:"category,omitempty"`
    Variants      []ProductVariant   `db:"-" json:"variants,omitempty"`
}

// ProductVariant represents a product color variant
type ProductVariant struct {
    ID              int       `db:"id" json:"id"`
    ProductID       int       `db:"product_id" json:"product_id"`
    Color           string    `db:"color" json:"color"`
    PhotoURL        string    `db:"photo_url" json:"photo_url"`
    PhotoID         string    `db:"photo_id" json:"photo_id"`
    PriceAdjustment float64   `db:"price_adjustment" json:"price_adjustment"`
    CreatedAt       time.Time `db:"created_at" json:"created_at"`
    UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// FinalPrice returns base_price + price_adjustment
func (v *ProductVariant) FinalPrice(basePrice float64) float64 {
    return basePrice + v.PriceAdjustment
}

// Category represents a product category
type Category struct {
    ID        int       `db:"id" json:"id"`
    Name      string    `db:"name" json:"name"`
    Slug      string    `db:"slug" json:"slug"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
    UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Admin represents an admin user
type Admin struct {
    ID           int       `db:"id" json:"id"`
    Username     string    `db:"username" json:"username"`
    PasswordHash string    `db:"password_hash" json:"-"` // Never expose in JSON
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
```

### Sample Data

```sql
-- Categories
INSERT INTO categories (name, slug) VALUES
    ('Kertas Bouquet', 'kertas-bouquet'),
    ('Pita & Ribbon', 'pita-ribbon'),
    ('Aksesoris Dekorasi', 'aksesoris-dekorasi'),
    ('Wrapping Material', 'wrapping-material');

-- Products
INSERT INTO products (code, title, description, category_id, base_price, is_sale) VALUES
    ('BRK-001', 'Kertas Bouquet Premium Gold', 'Kertas bouquet warna gold metalik, 50x50cm', 1, 50000, false),
    ('PTA-001', 'Pita Satin Pink 2cm', 'Pita satin halus warna pink, lebar 2cm, panjang 25m', 2, 30000, true);

-- Variants
INSERT INTO product_variants (product_id, color, price_adjustment) VALUES
    (1, 'Gold', 0),
    (1, 'Silver', -5000),
    (1, 'Rose Gold', 3000),
    (2, 'Pink', 0),
    (2, 'Baby Pink', 2000);

-- Admin (password: admin123)
INSERT INTO admins (username, password_hash) VALUES
    ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy');
```

### Data Validation Rules

**Product Code:**
- Pattern: `^[A-Z]{2,4}-[0-9]{3,5}$` (e.g., "BRK-001", "PITA-0042")
- Unique across all products
- Required

**Product Title:**
- Min: 5 characters
- Max: 200 characters
- Required

**Base Price:**
- Min: 0.01 (Rp 0.01)
- Max: 99,999,999.99
- Type: Decimal(12,2)
- Required

**Price Adjustment:**
- Can be negative (discount) or positive (premium)
- Type: Decimal(12,2)

**Photo Upload:**
- Formats: JPEG, PNG, WebP
- Max size: 5MB (before upload)
- Min dimensions: 300x300px
- Recommended: 1:1 aspect ratio (square)

**Category Slug:**
- Auto-generated from name
- Lowercase, hyphenated
- Pattern: `^[a-z0-9-]+$`

---

## 6. Environment Configuration

### Environment Variables

```bash
# Database (Auto-configured by Railway in production)
DATABASE_URL=postgresql://user:password@host:5432/dbname?sslmode=disable

# Cloudinary (Get from https://cloudinary.com/console)
CLOUDINARY_CLOUD_NAME=your-cloud-name
CLOUDINARY_API_KEY=your-api-key
CLOUDINARY_API_SECRET=your-api-secret

# Application
PORT=3000                    # Auto-set by Railway, default 3000 in dev
ENV=development              # development | production
JWT_SECRET=your-random-32-character-secret-key-here

# WhatsApp Integration
WHATSAPP_NUMBER=628123456789 # Seller's WhatsApp (with country code, no +)

# Optional: Default admin for seeding
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123      # Will be hashed before DB insert
```

### .env.example

```bash
# Copy this file to .env and fill in your values

# Database
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/flower_catalog?sslmode=disable

# Cloudinary - Sign up at https://cloudinary.com
CLOUDINARY_CLOUD_NAME=demo
CLOUDINARY_API_KEY=123456789012345
CLOUDINARY_API_SECRET=abcdefghijklmnopqrstuvwxyz12

# Application
PORT=3000
ENV=development
JWT_SECRET=dev-secret-change-this-in-production-use-random-32-chars

# WhatsApp
WHATSAPP_NUMBER=628123456789

# Admin (for seeding only)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
```

### Configuration Loading (Go)

```go
// internal/config/config.go
package config

import (
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    // Database
    DatabaseURL string
    
    // Cloudinary
    CloudName   string
    APIKey      string
    APISecret   string
    
    // Application
    Port        string
    Env         string
    JWTSecret   string
    
    // WhatsApp
    WhatsAppNumber string
}

func Load() *Config {
    // Load .env file (ignore error in production where env vars are set directly)
    _ = godotenv.Load()
    
    return &Config{
        DatabaseURL:    getEnv("DATABASE_URL", ""),
        CloudName:      getEnv("CLOUDINARY_CLOUD_NAME", ""),
        APIKey:         getEnv("CLOUDINARY_API_KEY", ""),
        APISecret:      getEnv("CLOUDINARY_API_SECRET", ""),
        Port:           getEnv("PORT", "3000"),
        Env:            getEnv("ENV", "development"),
        JWTSecret:      getEnv("JWT_SECRET", "dev-secret"),
        WhatsAppNumber: getEnv("WHATSAPP_NUMBER", ""),
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

// Validate checks if required env vars are set
func (c *Config) Validate() error {
    required := map[string]string{
        "DATABASE_URL":           c.DatabaseURL,
        "CLOUDINARY_CLOUD_NAME":  c.CloudName,
        "CLOUDINARY_API_KEY":     c.APIKey,
        "CLOUDINARY_API_SECRET":  c.APISecret,
        "JWT_SECRET":             c.JWTSecret,
    }
    
    for key, value := range required {
        if value == "" {
            return fmt.Errorf("required environment variable %s is not set", key)
        }
    }
    
    return nil
}
```

### Docker Environment Configuration

#### docker-compose.yml (Development)

```yaml
version: '3.9'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: flower_catalog
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d

  app:
    build:
      context: .
      dockerfile: docker/Dockerfile.dev
    ports:
      - "3000:3000"
    environment:
      DATABASE_URL: postgresql://postgres:postgres@postgres:5432/flower_catalog?sslmode=disable
      CLOUDINARY_CLOUD_NAME: ${CLOUDINARY_CLOUD_NAME}
      CLOUDINARY_API_KEY: ${CLOUDINARY_API_KEY}
      CLOUDINARY_API_SECRET: ${CLOUDINARY_API_SECRET}
      PORT: 3000
      ENV: development
      JWT_SECRET: dev-secret-change-in-production
      WHATSAPP_NUMBER: ${WHATSAPP_NUMBER}
    volumes:
      - .:/app
    depends_on:
      - postgres

volumes:
  postgres_data:
```

### Railway Environment Variables Setup

**Via Railway Dashboard:**
1. Go to your Railway project
2. Click on your service
3. Go to "Variables" tab
4. Add each variable:

```
DATABASE_URL          (auto-set by PostgreSQL plugin)
CLOUDINARY_CLOUD_NAME your-value
CLOUDINARY_API_KEY    your-value
CLOUDINARY_API_SECRET your-value
JWT_SECRET            generate-random-32-char-string
ENV                   production
PORT                  (auto-set by Railway)
WHATSAPP_NUMBER       628xxxxxxxxxx
```

**Generate JWT Secret:**
```bash
# In terminal
openssl rand -base64 32
# Or
head /dev/urandom | LC_ALL=C tr -dc 'A-Za-z0-9' | head -c 32
```

---

## 7. Security Best Practices

### Authentication & Authorization

#### Password Security

**Hashing Algorithm:** bcrypt (cost factor 10)

```go
// Hash password on registration/creation
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
    return string(bytes), err
}

// Verify password on login
func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**Password Requirements (MVP):**
- Min length: 8 characters
- No complexity requirements initially (avoid user friction)
- Can strengthen later if needed

**Never:**
- Store passwords in plain text
- Log passwords (even hashed)
- Send passwords in URLs or query params
- Display passwords in error messages

#### JWT Token Security

**Configuration:**
```go
type JWTClaims struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
    jwt.RegisteredClaims
}

// Generate token
func GenerateToken(userID int, username string, secret string) (string, error) {
    claims := JWTClaims{
        UserID:   userID,
        Username: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "flower-catalog",
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

// Verify token
func VerifyToken(tokenString string, secret string) (*JWTClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    
    if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, err
}
```

**Cookie Settings:**
```go
// Set token in HttpOnly cookie
c.Cookie(&fiber.Cookie{
    Name:     "auth_token",
    Value:    token,
    Path:     "/",
    MaxAge:   86400, // 24 hours
    HTTPOnly: true,  // Prevent XSS access
    Secure:   cfg.Env == "production", // HTTPS only in prod
    SameSite: "Strict", // CSRF protection
})
```

**Security Checklist:**
- ✅ Use strong secret (min 32 random characters)
- ✅ Rotate secret periodically (quarterly)
- ✅ HttpOnly cookies (prevent XSS)
- ✅ Secure flag in production (HTTPS only)
- ✅ SameSite=Strict (CSRF protection)
- ✅ Short expiry (24 hours max)
- ✅ Never log JWT tokens

#### Authentication Middleware

```go
// internal/middleware/auth.go
func AuthRequired(jwtSecret string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Get token from cookie
        token := c.Cookies("auth_token")
        if token == "" {
            return c.Redirect("/admin/login")
        }
        
        // Verify token
        claims, err := VerifyToken(token, jwtSecret)
        if err != nil {
            c.ClearCookie("auth_token")
            return c.Redirect("/admin/login")
        }
        
        // Store user info in context
        c.Locals("user_id", claims.UserID)
        c.Locals("username", claims.Username)
        
        return c.Next()
    }
}
```

### Input Validation & Sanitization

#### SQL Injection Prevention

**Always use parameterized queries (sqlx):**

```go
// ❌ DANGEROUS - SQL Injection vulnerability!
query := "SELECT * FROM products WHERE title LIKE '%" + userInput + "%'"
db.Query(query)

// ✅ SAFE - Parameterized query
query := "SELECT * FROM products WHERE title LIKE $1"
db.Query(query, "%"+userInput+"%")
```

**All database operations must use `$1, $2...` placeholders.**

#### XSS Prevention

**Go html/template auto-escapes:**
```html
<!-- User input: <script>alert('XSS')</script> -->
<p>{{.UserInput}}</p>
<!-- Rendered: <p>&lt;script&gt;alert('XSS')&lt;/script&gt;</p> -->
```

**For raw HTML (if needed):**
```go
// Use template.HTML only for trusted content
template.HTML("<b>Bold</b>") // Careful!

// Never use for user input
template.HTML(userInput) // ❌ NEVER DO THIS
```

**Content Security Policy (CSP) Headers:**
```go
app.Use(func(c *fiber.Ctx) error {
    c.Set("Content-Security-Policy", 
        "default-src 'self'; "+
        "script-src 'self' https://unpkg.com; "+
        "img-src 'self' https://res.cloudinary.com data:; "+
        "style-src 'self' 'unsafe-inline';")
    return c.Next()
})
```

#### CSRF Protection

**Fiber CSRF Middleware:**
```go
import "github.com/gofiber/fiber/v2/middleware/csrf"

app.Use(csrf.New(csrf.Config{
    KeyLookup:      "header:X-CSRF-Token",
    CookieName:     "csrf_",
    CookieSameSite: "Strict",
    Expiration:     1 * time.Hour,
}))
```

**In templates:**
```html
<form method="POST" action="/admin/products">
    <input type="hidden" name="_csrf" value="{{.CSRFToken}}">
    <!-- form fields -->
</form>
```

#### Rate Limiting

**Login endpoint protection:**
```go
import "github.com/gofiber/fiber/v2/middleware/limiter"

// Apply to login route
app.Post("/admin/login", limiter.New(limiter.Config{
    Max:        10,              // 10 attempts
    Expiration: 1 * time.Minute, // per minute
    KeyGenerator: func(c *fiber.Ctx) string {
        return c.IP() // Rate limit by IP
    },
    LimitReached: func(c *fiber.Ctx) error {
        return c.Status(429).JSON(fiber.Map{
            "error": "Too many login attempts. Try again in 1 minute.",
        })
    },
}), authHandler.Login)
```

### HTTPS & Secure Headers

#### HTTPS Enforcement

**Production (Railway):**
- Automatic HTTPS (Railway provides SSL cert)
- HTTP → HTTPS redirect (automatic)

**Development:**
- HTTP acceptable on localhost

**Helmet-like Security Headers:**
```go
app.Use(func(c *fiber.Ctx) error {
    c.Set("X-Content-Type-Options", "nosniff")
    c.Set("X-Frame-Options", "DENY")
    c.Set("X-XSS-Protection", "1; mode=block")
    c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
    
    if os.Getenv("ENV") == "production" {
        c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
    }
    
    return c.Next()
})
```

### Secrets Management

#### Never Commit Secrets to Git

**.gitignore:**
```
.env
.env.local
.env.production

# Sensitive files
*.pem
*.key
*.crt
```

#### Environment Variable Best Practices

**Development:**
- Use `.env` file (git-ignored)
- Provide `.env.example` (committed, no secrets)

**Production:**
- Set env vars in Railway dashboard
- Or use secrets manager (future: AWS Secrets Manager, Vault)

**Secret Rotation:**
- JWT_SECRET: Rotate quarterly
- Cloudinary API keys: Rotate if exposed
- Database password: Rotate annually

### Data Security

#### Sensitive Data Handling

**Never log:**
- Passwords (plain or hashed)
- JWT tokens
- API keys/secrets
- Credit card numbers (future if adding payments)

**Logging Best Practice:**
```go
// ❌ BAD
log.Printf("Login attempt: username=%s, password=%s", username, password)

// ✅ GOOD
log.Printf("Login attempt: username=%s", username)
```

**Database Encryption (Future):**
- For PII (personal identifiable information)
- Use PostgreSQL pgcrypto extension
- Not needed for MVP (no customer PII stored)

#### File Upload Security

**Cloudinary Upload Validation:**
```go
func (s *CloudinaryService) UploadProductImage(file multipart.File, filename string) (string, string, error) {
    // 1. Validate file type
    buffer := make([]byte, 512)
    _, err := file.Read(buffer)
    if err != nil {
        return "", "", err
    }
    file.Seek(0, 0) // Reset file pointer
    
    contentType := http.DetectContentType(buffer)
    allowed := []string{"image/jpeg", "image/png", "image/webp"}
    
    if !contains(allowed, contentType) {
        return "", "", errors.New("invalid file type")
    }
    
    // 2. Upload to Cloudinary (limits enforced by Cloudinary)
    resp, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
        Folder:         "flower-supply/products",
        PublicID:       filename,
        Transformation: "c_limit,w_1200,h_1200,q_auto,f_auto",
        ResourceType:   "image",
        MaxFileSize:    5242880, // 5MB
    })
    
    return resp.SecureURL, resp.PublicID, err
}
```

**File Upload Checklist:**
- ✅ Validate file type (magic number, not just extension)
- ✅ Enforce max file size (5MB)
- ✅ Use unique filenames (prevent overwrite)
- ✅ Upload to external storage (Cloudinary, not local disk)
- ✅ Scan for malware (future: ClamAV integration)

### Security Monitoring

#### Error Handling (Don't Leak Info)

```go
// ❌ BAD - Leaks stack trace
app.Use(func(c *fiber.Ctx) error {
    return c.Status(500).SendString(err.Error())
})

// ✅ GOOD - Generic message to user, detailed log for devs
func errorHandler(c *fiber.Ctx, err error) error {
    code := fiber.StatusInternalServerError
    message := "Internal Server Error"
    
    if e, ok := err.(*fiber.Error); ok {
        code = e.Code
        message = e.Message
    }
    
    // Log detailed error for debugging
    log.Printf("ERROR: %v", err)
    
    // Return generic message to user
    return c.Status(code).JSON(fiber.Map{
        "error": message,
    })
}
```

#### Audit Logging (Future)

**Log important actions:**
- Admin login/logout
- Product create/update/delete
- Image upload/delete
- Failed login attempts

**Log Format (JSON):**
```json
{
  "timestamp": "2026-01-15T10:30:00Z",
  "level": "INFO",
  "event": "product.created",
  "user_id": 1,
  "username": "admin",
  "product_id": 42,
  "product_code": "BRK-001",
  "ip_address": "192.168.1.100"
}
```

### Security Checklist

**Before Deployment:**
- [ ] All passwords hashed with bcrypt
- [ ] JWT secret is strong & unique (not "secret")
- [ ] HTTPS enabled in production
- [ ] HttpOnly cookies for JWT
- [ ] CSRF protection enabled
- [ ] Rate limiting on login
- [ ] Parameterized SQL queries (no string concatenation)
- [ ] Template auto-escaping enabled
- [ ] Security headers set
- [ ] No secrets in Git repository
- [ ] Error messages don't leak sensitive info
- [ ] File upload validation implemented
- [ ] Database connection encrypted (sslmode=require in prod)

**Regular Maintenance:**
- [ ] Update Go dependencies monthly
- [ ] Rotate JWT secret quarterly
- [ ] Review Railway logs for suspicious activity
- [ ] Monitor failed login attempts
- [ ] Check Cloudinary usage (avoid API key abuse)

---

**End of Simplified Project Context**

---
