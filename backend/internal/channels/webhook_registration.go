package channels

// WebhookRegistrationResult is returned by manual webhook registration from admin UI.
type WebhookRegistrationResult struct {
	Registered bool   `json:"registered"`
	Reason     string `json:"reason,omitempty"`
}
