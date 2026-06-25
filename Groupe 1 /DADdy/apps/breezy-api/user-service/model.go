package main

import "time"

// userModel est le profil détenu par le user-service. La PK est une string et
// vaut l'identifiant émis par l'auth-service (ex: "usr_ab12..."), celui que le
// gateway réinjecte dans le header X-User-Id. Le user-service ne génère donc
// jamais d'ID lui-même : il reçoit celui de l'auth au moment du register.
type userModel struct {
	ID          string  `gorm:"primaryKey;size:64"`
	Username    string  `gorm:"size:50;uniqueIndex;not null"`
	Email       string  `gorm:"size:255;uniqueIndex;not null"`
	DisplayName *string `gorm:"size:50"`
	// Pronouns en texte libre court (ex: "il/lui", "elle", "they/them") pour
	// laisser l'utilisateur s'exprimer sans liste figée.
	Pronouns  *string `gorm:"size:50"`
	Bio       *string `gorm:"size:160"`
	AvatarURL *string `gorm:"size:500"`
	// AvatarObjectKey est la clé de l'objet dans MinIO (ex: "avatars/usr_x/ab12.png").
	// Conservée pour pouvoir supprimer l'ancien fichier lors d'un remplacement.
	AvatarObjectKey *string `gorm:"size:255"`
	// BannerURL est l'image de bannière (en-tête du profil), uploadée comme l'avatar.
	BannerURL *string `gorm:"size:500"`
	// BannerObjectKey est la clé MinIO de la bannière (ex: "banners/usr_x/ab12.png"),
	// conservée pour supprimer l'ancien fichier lors d'un remplacement.
	BannerObjectKey *string `gorm:"size:255"`
	// IsPrivate passe le compte en privé : voir les posts/re-posts exige alors
	// d'être abonné, et s'abonner passe par une demande (follow_requests) que la
	// cible doit accepter. Les comptes publics conservent le comportement direct.
	IsPrivate      bool      `gorm:"not null;default:false"`
	Role           string    `gorm:"size:20;not null;default:user"`
	Status         string    `gorm:"size:20;not null;default:active"`
	FollowersCount int       `gorm:"not null;default:0"`
	FollowingCount int       `gorm:"not null;default:0"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (userModel) TableName() string { return "users" }

// followModel matérialise une relation d'abonnement orientée :
// FollowerID suit FolloweeID. La PK composite garantit l'idempotence.
type followModel struct {
	FollowerID string    `gorm:"primaryKey;size:64"`
	FolloweeID string    `gorm:"primaryKey;size:64;index"`
	CreatedAt  time.Time `gorm:"not null"`
}

func (followModel) TableName() string { return "follows" }

// followRequestModel matérialise une demande d'abonnement en attente vers un
// compte privé : RequesterID veut suivre TargetID. La PK composite garantit
// l'idempotence. À l'acceptation, la ligne est remplacée par un followModel ;
// au refus (ou à l'annulation), elle est simplement supprimée.
type followRequestModel struct {
	RequesterID string    `gorm:"primaryKey;size:64"`
	TargetID    string    `gorm:"primaryKey;size:64;index"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (followRequestModel) TableName() string { return "follow_requests" }

// sanctionModel matérialise une sanction de modération (suspension ou ban)
// appliquée à un utilisateur. EndedAt nil = sanction active ; renseigné lorsque
// la sanction est levée. Le statut courant du compte vit sur userModel.Status.
type sanctionModel struct {
	ID          int64      `gorm:"primaryKey;autoIncrement"`
	UserID      string     `gorm:"size:64;not null;index:idx_sanction_user"`
	ModeratorID *string    `gorm:"size:64"`
	Type        string     `gorm:"size:20;not null"`
	Reason      *string    `gorm:"size:255"`
	StartedAt   time.Time  `gorm:"not null;default:now()"`
	EndedAt     *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"not null;default:now()"`
}

func (sanctionModel) TableName() string { return "sanctions" }

// blockModel matérialise un blocage : BlockerID a bloqué BlockedID.
type blockModel struct {
	BlockerID string    `gorm:"primaryKey;size:64"`
	BlockedID string    `gorm:"primaryKey;size:64;index"`
	CreatedAt time.Time `gorm:"not null"`
}

func (blockModel) TableName() string { return "blocks" }
