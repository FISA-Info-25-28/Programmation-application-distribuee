package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID             string    `gorm:"primaryKey;size:64"`
	Username       string    `gorm:"size:50;not null;uniqueIndex"`
	Email          string    `gorm:"size:255;not null;uniqueIndex"`
	Password       string    `gorm:"size:255;not null" json:"-"`
	Bio            *string   `gorm:"type:text"`
	ProfilePicture *string   `gorm:"size:255"`
	Role           string    `gorm:"size:20;not null;default:'user';check:chk_user_role,role IN ('user','moderator','administrator')"`
	Status         string    `gorm:"size:20;not null;default:'active';check:chk_user_status,status IN ('active','suspended','banned')"`
	Language       string    `gorm:"size:5;not null;default:'en'"`
	Theme          string    `gorm:"size:20;not null;default:'light'"`
	FollowersCount int       `gorm:"not null;default:0"`
	FollowingCount int       `gorm:"not null;default:0"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
	UpdatedAt      time.Time `gorm:"not null;default:now()"`
}

func (User) TableName() string { return "users" }

type Post struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	Content       string    `gorm:"size:280;not null"`
	AuthorID      string    `gorm:"size:64;not null;index"`
	Author        User      `gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`
	ParentID      *int64    `gorm:"index:idx_post_parent"`
	Parent        *Post     `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	LikesCount    int       `gorm:"not null;default:0"`
	CommentsCount int       `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

func (Post) TableName() string { return "posts" }

type PostTag struct {
	PostID int64  `gorm:"primaryKey"`
	Tag    string `gorm:"primaryKey;size:100"`
	Post   Post   `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (PostTag) TableName() string { return "post_tags" }

type Comment struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	PostID          int64     `gorm:"not null;index"`
	ParentCommentID *int64    `gorm:"index"`
	AuthorID        string    `gorm:"size:64;not null;index"`
	Content         string    `gorm:"type:text;not null"`
	CreatedAt       time.Time `gorm:"not null;default:now()"`
	Post            Post      `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Author          User      `gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`
}

func (Comment) TableName() string { return "comments" }

type Like struct {
	UserID    string    `gorm:"primaryKey;size:64"`
	PostID    int64     `gorm:"primaryKey;index:idx_like_post"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Post      Post      `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (Like) TableName() string { return "likes" }

type Follow struct {
	FollowerID string    `gorm:"primaryKey;size:64"`
	FolloweeID string    `gorm:"primaryKey;size:64;index:idx_follow_followee"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
	Follower   User      `gorm:"foreignKey:FollowerID;references:ID;constraint:OnDelete:CASCADE"`
	Followee   User      `gorm:"foreignKey:FolloweeID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Follow) TableName() string { return "follows" }

type Hashtag struct {
	ID   int64  `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"size:100;not null;uniqueIndex"`
}

func (Hashtag) TableName() string { return "hashtags" }

type PostHashtag struct {
	PostID    int64   `gorm:"primaryKey"`
	HashtagID int64   `gorm:"primaryKey;index:idx_post_hashtag_hashtag"`
	Post      Post    `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Hashtag   Hashtag `gorm:"foreignKey:HashtagID;constraint:OnDelete:CASCADE"`
}

func (PostHashtag) TableName() string { return "post_hashtags" }

type Media struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	PostID    int64     `gorm:"not null;index:idx_media_post"`
	MediaType string    `gorm:"size:10;not null;check:chk_media_type,media_type IN ('image','video')"`
	URL       string    `gorm:"size:255;not null"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	Post      Post      `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (Media) TableName() string { return "media" }

type Mention struct {
	PostID int64  `gorm:"primaryKey"`
	UserID string `gorm:"primaryKey;size:64"`
	Post   Post   `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	User   User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (Mention) TableName() string { return "mentions" }

type Notification struct {
	ID          int64           `gorm:"primaryKey;autoIncrement"`
	RecipientID string          `gorm:"size:64;not null;index:idx_notification_recipient"`
	Type        string          `gorm:"size:20;not null;check:chk_notification_type,type IN ('like','follow','mention','reply')"`
	Data        json.RawMessage `gorm:"type:jsonb"`
	Read        bool            `gorm:"not null;default:false"`
	CreatedAt   time.Time       `gorm:"not null;default:now();index:idx_notification_recipient,sort:desc"`
	Recipient   User            `gorm:"foreignKey:RecipientID;constraint:OnDelete:CASCADE"`
}

func (Notification) TableName() string { return "notifications" }

type PrivateMessage struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	SenderID    string    `gorm:"size:64;not null;index:idx_private_message_conversation"`
	RecipientID string    `gorm:"size:64;not null;index:idx_private_message_conversation"`
	Content     string    `gorm:"type:text;not null"`
	Read        bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"not null;default:now();index:idx_private_message_conversation"`
	Sender      User      `gorm:"foreignKey:SenderID;references:ID;constraint:OnDelete:CASCADE"`
	Recipient   User      `gorm:"foreignKey:RecipientID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PrivateMessage) TableName() string { return "private_messages" }

type Report struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ReporterID  string    `gorm:"size:64;not null"`
	PostID      int64     `gorm:"not null"`
	ModeratorID *string   `gorm:"size:64"`
	Reason      string    `gorm:"size:255;not null"`
	Status      string    `gorm:"size:20;not null;default:'pending';check:chk_report_status,status IN ('pending','processed','rejected')"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	Reporter    User      `gorm:"foreignKey:ReporterID;references:ID;constraint:OnDelete:CASCADE"`
	Post        Post      `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Moderator   *User     `gorm:"foreignKey:ModeratorID;references:ID;constraint:OnDelete:SET NULL"`
}

func (Report) TableName() string { return "reports" }

type Sanction struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	UserID      string    `gorm:"size:64;not null"`
	ModeratorID *string   `gorm:"size:64"`
	Type        string    `gorm:"size:20;not null;check:chk_sanction_type,type IN ('suspension','ban')"`
	Reason      *string   `gorm:"size:255"`
	StartedAt   time.Time `gorm:"not null;default:now()"`
	EndedAt     *time.Time
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	User        User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Moderator   *User     `gorm:"foreignKey:ModeratorID;references:ID;constraint:OnDelete:SET NULL"`
}

func (Sanction) TableName() string { return "sanctions" }
