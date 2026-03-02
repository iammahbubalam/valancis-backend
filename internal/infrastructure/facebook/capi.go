package facebook

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// HashSHA256 returns a hex-encoded SHA256 hash of the normalized input string.
func HashSHA256(input string) string {
	if input == "" {
		return ""
	}
	// Normalize: trim whitespace and lowercase
	normalized := strings.ToLower(strings.TrimSpace(input))
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// CAPIClient handles server-side event tracking to Facebook Conversions API
type CAPIClient struct {
	pixelID     string
	accessToken string
	apiVersion  string
	httpClient  *http.Client
}

// NewCAPIClient creates a new Facebook CAPI client
func NewCAPIClient(pixelID, accessToken, apiVersion string) *CAPIClient {
	if pixelID == "" || accessToken == "" {
		log.Println("[CAPI] Facebook Pixel ID or Access Token not configured. CAPI disabled.")
		return nil
	}
	return &CAPIClient{
		pixelID:     pixelID,
		accessToken: accessToken,
		apiVersion:  apiVersion,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// UserData represents the user information for event matching
type UserData struct {
	Email      string `json:"em,omitempty"`          // SHA256 hashed email
	Phone      string `json:"ph,omitempty"`          // SHA256 hashed phone
	FirstName  string `json:"fn,omitempty"`          // SHA256 hashed first name
	LastName   string `json:"ln,omitempty"`          // SHA256 hashed last name
	City       string `json:"ct,omitempty"`          // SHA256 hashed city
	State      string `json:"st,omitempty"`          // SHA256 hashed state/region
	Zip        string `json:"zp,omitempty"`          // SHA256 hashed zip/postal code
	Country    string `json:"country,omitempty"`     // ISO 2-letter country code
	ExternalID string `json:"external_id,omitempty"` // Any unique ID from your system
	ClientIP   string `json:"client_ip_address,omitempty"`
	UserAgent  string `json:"client_user_agent,omitempty"`
	FBC        string `json:"fbc,omitempty"` // Facebook Click ID from _fbc cookie
	FBP        string `json:"fbp,omitempty"` // Facebook Browser ID from _fbp cookie
}

// CustomData represents purchase-specific data
type CustomData struct {
	Currency    string        `json:"currency,omitempty"`
	Value       float64       `json:"value,omitempty"`
	ContentName string        `json:"content_name,omitempty"`
	ContentIDs  []string      `json:"content_ids,omitempty"`
	Contents    []ContentItem `json:"contents,omitempty"`
	NumItems    int           `json:"num_items,omitempty"`
	OrderID     string        `json:"order_id,omitempty"`
}

// ContentItem represents individual product in the order
type ContentItem struct {
	ID       string  `json:"id"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"item_price,omitempty"`
}

// Event represents a single CAPI event
type Event struct {
	EventName      string     `json:"event_name"`
	EventTime      int64      `json:"event_time"`
	ActionSource   string     `json:"action_source"`
	EventSourceURL string     `json:"event_source_url,omitempty"`
	UserData       UserData   `json:"user_data"`
	CustomData     CustomData `json:"custom_data,omitempty"`
	EventID        string     `json:"event_id,omitempty"` // For deduplication with browser events
}

// EventPayload is the request body for CAPI
type EventPayload struct {
	Data []Event `json:"data"`
}

// SendEvent sends a single event to Facebook CAPI with simple retry logic
func (c *CAPIClient) SendEvent(event Event) error {
	if c == nil {
		return nil // CAPI disabled
	}

	payload := EventPayload{
		Data: []Event{event},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/events?access_token=%s",
		c.apiVersion, c.pixelID, c.accessToken)

	var lastErr error
	for i := 0; i < 3; i++ { // Retry up to 3 times
		resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("CAPI request failed: %w", err)
			time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoffish
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("[CAPI] Event '%s' sent successfully", event.EventName)
			return nil
		}

		body, _ := io.ReadAll(resp.Body)
		lastErr = fmt.Errorf("CAPI error (status %d): %s", resp.StatusCode, string(body))

		// If it's a 4xx error (other than 429), don't retry as it's likely a permanent error in payload
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return lastErr
}

// SendPurchaseEvent is a convenience method for purchase events
func (c *CAPIClient) SendPurchaseEvent(orderID string, value float64, currency string, items []ContentItem, userData UserData, eventID string) {
	if c == nil {
		return
	}

	// PII Hashing: Standardize and hash sensitive fields
	userData.Email = HashSHA256(userData.Email)
	userData.Phone = HashSHA256(userData.Phone)
	userData.FirstName = HashSHA256(userData.FirstName)
	userData.LastName = HashSHA256(userData.LastName)
	userData.City = HashSHA256(userData.City)
	userData.State = HashSHA256(userData.State)
	userData.Zip = HashSHA256(userData.Zip)

	event := Event{
		EventName:    "Purchase",
		EventTime:    time.Now().Unix(),
		ActionSource: "website",
		UserData:     userData,
		CustomData: CustomData{
			Currency:   currency,
			Value:      value,
			OrderID:    orderID,
			Contents:   items,
			NumItems:   len(items),
			ContentIDs: extractContentIDs(items),
		},
		EventID: eventID, // Use order ID for deduplication
	}

	// Send async to not block the order flow
	go func() {
		if err := c.SendEvent(event); err != nil {
			log.Printf("[CAPI] Failed to send Purchase event: %v", err)
		}
	}()
}

func extractContentIDs(items []ContentItem) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}
