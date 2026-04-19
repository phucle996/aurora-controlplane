package iam_reqdto

// RefreshTokenRequest is the client payload for obtaining a new token pair.
//
// The client MUST sign the canonical payload:
//
//	SHA256( refresh_token + "." + device_id + "." + nonce + "." + timestamp_unix )
//
// with its device private key and base64-raw-url encode the resulting bytes
// as the `signature` field.
type RefreshTokenRequest struct {
	// Nonce is a client-chosen random string (min 16 chars) to defeat replays.
	Nonce string `json:"nonce" binding:"required,min=16"`

	// TimestampUnix is Unix epoch seconds at signing time.
	// Must be within ±5 minutes of server time.
	TimestampUnix int64 `json:"timestamp" binding:"required"`

	// Signature is the base64-raw-url encoded device signature over
	// the canonical payload.
	Signature string `json:"signature" binding:"required"`
}
