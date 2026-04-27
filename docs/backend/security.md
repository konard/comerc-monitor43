# Security & Compliance

## 152-ФЗ Compliance

### Data Residency

**Требование:** Персональные данные должны храниться на территории РФ

**Решение:**
- **Cloud Provider:** Timeweb Cloud (Российский провайдер)
- **Data Centers:** Москва, Санкт-Петербург
- **Database:** PostgreSQL в Timeweb Cloud
- **Backups:** Хранятся в том же регионе

### Data Encryption

#### In-Transit (TLS 1.3)
```go
// Force TLS 1.3
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS13,
    MaxVersion: tls.VersionTLS13,
}
```

#### At-Rest (Disk/DB)
```yaml
# PostgreSQL encryption
postgresql:
  encryption: true
  algorithm: AES-256
```

#### Field-Level (PDE)
```go
import "crypto/aes"

func EncryptField(plaintext []byte, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    // GCM mode for authenticated encryption
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}
```

**Encrypted fields:**
- `users.phone_encrypted`
- `users.email` (опционально)
- payment details

---

## API Security

### Authentication

#### Stateless JWT

```go
import "github.com/golang-jwt/jwt"

type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Tier   string `json:"tier"`
    jwt.RegisteredClaims
}

func GenerateToken(userID, email, tier string) (string, error) {
    claims := Claims{
        UserID: userID,
        Email:  email,
        Tier:   tier,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "milon",
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(jwtSecret))
}
```

#### Token Rotation

```
Access Token:  15 минут
Refresh Token: 7 дней
```

---

## System Architecture

### Single-User Model with Tier-Based Limits

The system implements a simplified access model:

- **Authentication**: Users authenticate via JWT tokens
- **Tier-Based Limits**: Resource limits determined by subscription tier (Free, Starter, Professional, Business)
- **Resource Isolation**: Users can only access their own resources (enforced via user_id)
- **No Role Distinctions**: All authenticated users have uniform feature access

### Tier Permissions

Tier-based limits define resource quotas:
- **Free**: 25 monitors, 5min check interval, 3 alert channels
- **Starter**: 100 monitors, 1min check interval, 10 alert channels
- **Professional**: 500 monitors, 30sec check interval, 50 alert channels
- **Business**: Unlimited monitors, 10sec check interval, unlimited alert channels

### Rate Limiting

#### In-Memory Rate Limiter

```go
import "go.uber.org/ratelimit"

type RateLimiter struct {
    limiters map[string]ratelimit.Limiter
    mu       sync.RWMutex
}

func (r *RateLimiter) GetLimiter(userID string) ratelimit.Limiter {
    r.mu.RLock()
    limiter, exists := r.limiters[userID]
    r.mu.RUnlock()

    if !exists {
        r.mu.Lock()
        limiter = ratelimit.New(100) // 100 req/sec
        r.limiters[userID] = limiter
        r.mu.Unlock()
    }

    return limiter
}
```

#### Per-Tier Limits

```go
var TierRateLimits = map[string]int{
    "free":         10,   // 10 req/sec
    "starter":      50,   // 50 req/sec
    "professional": 100,  // 100 req/sec
    "business":     1000, // 1000 req/sec
}
```

---

## Input Validation

### API Level

```go
type CreateMonitorRequest struct {
    Name     string `json:"name" validate:"required,min=1,max=255"`
    URL      string `json:"url" validate:"required,url"`
    Interval int    `json:"interval" validate:"required,min=30,max=1800"`
}

func (r *CreateMonitorRequest) Validate() error {
    if r.Interval < 30 || r.Interval > 1800 {
        return errors.New("interval must be between 30 and 1800 seconds")
    }

    if _, err := url.ParseRequestURI(r.URL); err != nil {
        return err
    }

    return nil
}
```

### Database Level

```sql
-- Constraints
ALTER TABLE monitors
ADD CONSTRAINT check_interval_range
CHECK (interval_seconds IN (30, 60, 120, 300, 600, 1800));

ALTER TABLE monitors
ADD CONSTRAINT check_url_format
CHECK (url ~* '^https?://');
```

---

## Password Security

### Hashing

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### Password Requirements

```go
func ValidatePassword(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }

    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

    if !hasUpper || !hasLower || !hasDigit {
        return errors.New("password must contain uppercase, lowercase, and digit")
    }

    return nil
}
```

---

## Webhook Security

### Signature Verification

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func VerifyWebhook(signature string, payload []byte, secret string) bool {
    expectedMAC := computeHMAC(payload, secret)
    return hmac.Equal([]byte(signature), expectedMAC)
}

func computeHMAC(message []byte, secret string) []byte {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(message)
    return []byte(hex.EncodeToString(h.Sum(nil)))
}
```

### User Webhooks

```go
type WebhookConfig struct {
    URL         string            `json:"url"`
    Method      string            `json:"method"`
    Headers     map[string]string `json:"headers"`
    Signature   string            `json:"signature"`
    Timeout     int               `json:"timeout"`
}

func (w *WebhookConfig) Send(payload []byte) error {
    req, err := http.NewRequest(w.Method, w.URL, bytes.NewReader(payload))
    if err != nil {
        return err
    }

    // Add custom headers
    for k, v := range w.Headers {
        req.Header.Set(k, v)
    }

    // Add signature
    sig := computeHMAC(payload, w.Signature)
    req.Header.Set("X-Milon-Signature", string(sig))

    client := &http.Client{Timeout: time.Duration(w.Timeout) * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("webhook returned status %d", resp.StatusCode)
    }

    return nil
}
```

---

## Payment Security (PCI DSS)

### Yandex Kassa Integration

#### Compliance
- ✅ Не храним карточные данные
- ✅ Используем tokenization от ЮKassa
- ✅ PCI DSS compliant провайдер

#### Webhook Verification

```go
func VerifyKassaWebhook(signature string, payload []byte, secret string) bool {
    // ЮKassa uses HMAC-SHA256
    expectedSignature := computeHMAC(payload, secret)
    return hmac.Equal([]byte(signature), expectedSignature)
}
```

#### Idempotency Keys

```go
func CreatePayment(user User, amount int) (*Payment, error) {
    idempotencyKey := uuid.New().String()

    req := PaymentRequest{
        Amount:        amount,
        Currency:      "RUB",
        IdempotencyKey: idempotencyKey,
    }

    // Store idempotency key
    if err := s.db.StoreIdempotencyKey(idempotencyKey); err != nil {
        return nil, err
    }

    return s.kassa.CreatePayment(req)
}
```

---

## Logging Security

### Personal Data Redaction

```go
import "regexp"

func SanitizeLog(log string) string {
    // Redact email addresses
    emailRegex := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
    log = emailRegex.ReplaceAllString(log, "***@***.***")

    // Redact phone numbers
    phoneRegex := regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{2}[-.\s]?\d{2}\b`)
    log = phoneRegex.ReplaceAllString(log, "+7 *** *** ** **")

    // Redact IP addresses (optional)
    ipRegex := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
    log = ipRegex.ReplaceAllString(log, "***.***.***.***")

    return log
}
```

### Log Retention (152-ФЗ)

```
Free: 7 дней
Starter: 30 дней
Professional: 90 дней
Business: 1 год
```

---

## Network Security

### Firewall Rules

```bash
# Allow only necessary ports
# K8s API server
ufw allow from 10.0.0.0/8 to any port 6443

# HTTP/HTTPS
ufw allow 80/tcp
ufw allow 443/tcp

# Deny everything else
ufw default deny incoming
ufw default allow outgoing
```

### Pod Security Policies

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: milan-restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
```

---

## Dependency Scanning

### Go Dependency Checks

```bash
# Check for vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Static analysis
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

### CI/CD Integration

```yaml
security-scan:
  stage: test
  script:
    - govulncheck ./...
    - staticcheck ./...
    - go test -race ./...
```

---

## Incident Response

### Security Incident Checklist

1. **Identification**
   - [ ] Мониторинг алертов
   - [ ] Log analysis
   - [ ] User reports

2. **Containment**
   - [ ] Изолировать affected systems
   - [ ] Block malicious IPs
   - [ ] Disable compromised accounts

3. **Eradication**
   - [ ] Remove malware
   - [ ] Patch vulnerabilities
   - [ ] Update dependencies

4. **Recovery**
   - [ ] Restore from backups
   - [ ] Monitor for recurrence
   - [ ] Document lessons learned

5. **Reporting**
   - [ ] Notify affected users (152-ФЗ requirement)
   - [ ] Report to authorities (if required)
   - [ ] Update policies

---

**Дата:** 2026-03-08
**Версия:** 1.0
