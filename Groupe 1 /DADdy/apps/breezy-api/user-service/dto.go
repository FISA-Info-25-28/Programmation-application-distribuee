package main

// userResponse suit le schéma `User` de l'OpenAPI (camelCase).
//   - email : pointeur + omitempty → présent uniquement quand l'appelant
//     consulte son propre profil.
//   - isFollowedByMe : pointeur + omitempty → présent uniquement sur GET /users/{id}.
//   - bio / avatarUrl : sans omitempty → exposés en null si absents (nullable).
type userResponse struct {
	ID             string  `json:"id"`
	Username       string  `json:"username"`
	Email          *string `json:"email,omitempty"`
	DisplayName    *string `json:"displayName"`
	Pronouns       *string `json:"pronouns"`
	Bio            *string `json:"bio"`
	AvatarURL      *string `json:"avatarUrl"`
	BannerURL      *string `json:"bannerUrl"`
	Role           string  `json:"role"`
	IsPrivate      bool    `json:"isPrivate"`
	Status         string  `json:"status"`
	FollowersCount int     `json:"followersCount"`
	FollowingCount int     `json:"followingCount"`
	IsFollowedByMe *bool   `json:"isFollowedByMe,omitempty"`
	// FollowRequested : présent sur GET /users/{id}, true si l'appelant a une
	// demande d'abonnement en attente vers ce compte privé.
	FollowRequested *bool `json:"followRequested,omitempty"`
	// CanViewPosts : présent sur GET /users/{id}, indique si l'appelant peut voir
	// les posts/re-posts (soi-même, compte public, ou abonné d'un compte privé).
	CanViewPosts  *bool  `json:"canViewPosts,omitempty"`
	IsBlockedByMe *bool  `json:"isBlockedByMe,omitempty"`
	HasBlockedMe  *bool  `json:"hasBlockedMe,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

// updateProfileRequest ne porte plus l'avatar : celui-ci est désormais un fichier
// image uploadé via PUT /users/me/avatar, et non plus une URL libre.
type updateProfileRequest struct {
	DisplayName *string `json:"displayName"`
	Pronouns    *string `json:"pronouns"`
	Bio         *string `json:"bio"`
	// IsPrivate : pointeur pour ne mettre à jour la confidentialité que lorsque le
	// champ est explicitement fourni.
	IsPrivate *bool `json:"isPrivate"`
}

// avatarResponse renvoyé après un upload d'avatar (PUT /users/me/avatar).
type avatarResponse struct {
	AvatarURL string `json:"avatarUrl"`
}

// bannerResponse renvoyé après un upload de bannière (PUT /users/me/banner).
type bannerResponse struct {
	BannerURL string `json:"bannerUrl"`
}

// sanctionRequest est le corps de POST /users/:id/sanctions (staff).
type sanctionRequest struct {
	Type   string  `json:"type"`
	Reason *string `json:"reason"`
}

// sanctionResponse suit le schéma exposé au front (camelCase).
type sanctionResponse struct {
	ID          int64   `json:"id"`
	UserID      string  `json:"userId"`
	ModeratorID *string `json:"moderatorId"`
	Type        string  `json:"type"`
	Reason      *string `json:"reason"`
	StartedAt   string  `json:"startedAt"`
	EndedAt     *string `json:"endedAt"`
	CreatedAt   string  `json:"createdAt"`
}

// internalCreateUserRequest est le payload de POST /internal/users, envoyé par
// l'auth-service au moment du register. Non exposé via le gateway.
type internalCreateUserRequest struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}
