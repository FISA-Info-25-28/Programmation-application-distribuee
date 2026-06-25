package main

type registerRequest struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	TermsAccepted bool   `json:"termsAccepted"`
	TermsVersion  string `json:"termsVersion"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Email      string `json:"email"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type resendVerificationRequest struct {
	Email string `json:"email"`
}

type requestPasswordResetRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type authResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type mfaRequiredResponse struct {
	MFARequired  bool   `json:"mfaRequired"`
	PendingToken string `json:"pendingToken"`
}

type mfaSetupResponse struct {
	Secret      string   `json:"secret"`
	URI         string   `json:"uri"`
	BackupCodes []string `json:"backupCodes"`
}

type mfaVerifyRequest struct {
	PendingToken string `json:"pendingToken"`
	Code         string `json:"code"`
}

type mfaConfirmRequest struct {
	Code string `json:"code"`
}

type mfaDisableRequest struct {
	Code string `json:"code"`
}

type messageResponse struct {
	Message string `json:"message"`
}

type meResponse struct {
	ID             string  `json:"id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	Language       string  `json:"language"`
	Theme          string  `json:"theme"`
	Bio            *string `json:"bio"`
	AvatarUrl      *string `json:"avatarUrl"`
	FollowersCount int     `json:"followersCount"`
	FollowingCount int     `json:"followingCount"`
	MFAEnabled     bool    `json:"mfaEnabled"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}
