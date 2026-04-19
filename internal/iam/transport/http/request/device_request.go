package iam_reqdto

// ── Security ──────────────────────────────────────────────────────────────────

type IssueChallengeRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

type VerifyProofRequest struct {
	ChallengeID string `json:"challenge_id" binding:"required"`
	DeviceID    string `json:"device_id" binding:"required"`
	Signature   string `json:"signature" binding:"required"`
}

type RotateKeyRequest struct {
	DeviceID     string `json:"device_id" binding:"required"`
	NewPublicKey string `json:"new_public_key" binding:"required"`
	NewAlgorithm string `json:"new_algorithm" binding:"required"`
}

type RebindRequest struct {
	DeviceID     string `json:"device_id" binding:"required"`
	ChallengeID  string `json:"challenge_id" binding:"required"`
	Signature    string `json:"signature" binding:"required"`
	NewPublicKey string `json:"new_public_key" binding:"required"`
	NewAlgorithm string `json:"new_algorithm" binding:"required"`
}

// ── Self-service ──────────────────────────────────────────────────────────────

type RenameDeviceRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
}

// ── Admin ─────────────────────────────────────────────────────────────────────

type AdminMarkSuspiciousRequest struct {
	Suspicious bool `json:"suspicious"`
}

type CleanupStaleRequest struct {
	// InactiveDays is the number of days of inactivity to consider a device stale.
	InactiveDays int `json:"inactive_days" binding:"required,min=1"`
}
