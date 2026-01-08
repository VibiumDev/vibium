package appium

// Capability represents a browser or mobile capability
type Capability map[string]interface{}

// Capabilities represents the desired capabilities for a session
type Capabilities struct {
	AlwaysMatch Capability `json:"alwaysMatch"`
	FirstMatch  []Capability `json:"firstMatch,omitempty"`
}

// NewSessionRequest represents the payload to start a new session
type NewSessionRequest struct {
	Capabilities Capabilities `json:"capabilities"`
}

// Session represents an active Appium session
type Session struct {
	ID           string                 `json:"sessionId"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

// Response represents a standard WebDriver JSON response
type Response struct {
	Value interface{} `json:"value"`
}

// Element represents a UI element (finding returns this)
type Element struct {
	ID string `json:"element-6066-11e4-a52e-4f735466cecf"`
}

// Rect represents element dimensions
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
