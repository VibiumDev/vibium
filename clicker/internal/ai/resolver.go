package ai

import "context"

// Request is the input to a Resolver. Image is optional — nil means text-only.
type Request struct {
	Prompt string
	Image  []byte // PNG bytes; nil for text-only requests
}

// Response is the output from a Resolver.
type Response struct {
	Text string
}

// Resolver resolves a natural language prompt (with optional screenshot) to a text response.
// Implementations must be safe for concurrent use.
type Resolver interface {
	Resolve(ctx context.Context, req Request) (Response, error)
}
