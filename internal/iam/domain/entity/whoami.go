package entity

// WhoAmI is the flattened authenticated session view returned by /whoami.
type WhoAmI struct {
	UserID         string
	Username       string
	Email          string
	Phone          string
	FullName       string
	Company        string
	ReferralSource string
	JobFunction    string
	Country        string
	AvatarURL      string
	Bio            string
	Status         string
	OnBoarding     bool
	Level          int16
	AuthType       string
	SessionID      string
	Roles          []string
	Permissions    []string
}
