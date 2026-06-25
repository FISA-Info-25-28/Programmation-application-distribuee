package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	neturl "net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"daddy/apps/breezy-api/internal/shared"
)

// â"€â"€ Constants â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€

const (
	mediaTypeImage = "image"
	mediaTypeVideo = "video"
	mediaTypeAudio = "audio"

	extJPG  = ".jpg"
	extMP4  = ".mp4"
	extWebM = ".webm"

	mimeOctetStream  = "application/octet-stream"
	mimeJPEG         = "image/jpeg"
	mimeMP4          = "video/mp4"
	mimeQuicktime    = "video/quicktime"
	mimeMPEG         = "audio/mpeg"
	mimeOGG          = "audio/ogg"
	keyError         = "error"
	keyData          = "data"
	keyPage          = "page"
	keyLimit         = "limit"
	keyTotal         = "total"
	msgFetchPost     = "failed to fetch post"
	msgFetchPosts    = "failed to fetch posts"
	msgInvalidBody   = "invalid request body"
	msgFileRequired  = "file field required"
	defaultMinioUser = "minioadmin"
	maxMediaSizeText = "file must not exceed 300 MiB"
	maxMediaSize     = 300 << 20
	maxMultipartSize = maxMediaSize + (1 << 20)
	maxMultipartRAM  = 32 << 20
	mediaPartSize    = 32 << 20

	notifKeyUserID     = "user_id"
	notifKeyType       = "type"
	notifKeyActorID    = "actor_id"
	notifKeyActorUser  = "actor_username"
	notifKeyEntityID   = "entity_id"
	notifKeyEntityType = "entity_type"
	notifEntityPost    = "post"
	keyMessage         = "message"
	colLikesCount      = "likes_count"
	colRebreezeCount   = "rebreeze_count"
	notifTypeMention   = "mention"
	notifTypeLike      = "like"
	notifTypeRebreeze  = "rebreeze"
)

var (
	errMediaTooLarge     = errors.New("media file too large")
	errMediaFileRequired = errors.New("media file required")
	errUnsupportedMedia  = errors.New("unsupported media type")
)

var errInvalidMediaIDs = errors.New("invalid media ids")

// â"€â"€ Models â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€

type postModel struct {
	ID             int64      `gorm:"primaryKey;autoIncrement"`
	Content        string     `gorm:"size:280;not null"`
	AuthorID       string     `gorm:"size:64;not null;index:idx_post_author"`
	AuthorUsername string     `gorm:"size:50"`
	ParentID       *int64     `gorm:"index:idx_post_parent"`
	Parent         *postModel `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	QuoteID        *int64     `gorm:"index:idx_post_quote"`
	Quote          *postModel `gorm:"foreignKey:QuoteID;constraint:OnDelete:SET NULL"`
	LikesCount     int        `gorm:"not null;default:0"`
	CommentsCount  int        `gorm:"not null;default:0"`
	RebreezeCount  int        `gorm:"not null;default:0"`
	// Theme est le thème du breeze, jamais choisi par l'auteur : il émerge des
	// données — cluster des hashtags du post (cf. clusters.go), ou à défaut
	// comportement des interactants (cf. theme_infer.go). Sa valeur est le label
	// d'un cluster (= un hashtag central), pas un nom figé. NULL tant qu'aucun
	// thème ne ressort. Signal de préférence pour l'algorithme Pour Toi.
	Theme *string `gorm:"size:100;index:idx_post_theme"`
	// ThemeInferred vaut true quand le thème a été déduit du comportement des
	// interactants (et non des hashtags) ; ces thèmes restent révisables, alors que
	// ceux issus des hashtags sont définitifs.
	ThemeInferred bool      `gorm:"not null;default:false"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

func (postModel) TableName() string { return "posts" }

type likeModel struct {
	UserID    string    `gorm:"primaryKey;size:64"`
	PostID    int64     `gorm:"primaryKey;index:idx_like_post"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	Post      postModel `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (likeModel) TableName() string { return "likes" }

type rebreezersModel struct {
	UserID    string    `gorm:"primaryKey;size:64"`
	PostID    int64     `gorm:"primaryKey;index:idx_rebreezer_post"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	Post      postModel `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (rebreezersModel) TableName() string { return "rebreezers" }

// bookmarkModel matérialise un signet (post sauvegardé) d'un utilisateur. Privé :
// aucun compteur public sur le post, aucune notification à l'auteur. La clé
// primaire (user_id, post_id) empêche les doublons.
type bookmarkModel struct {
	UserID    string    `gorm:"primaryKey;size:64"`
	PostID    int64     `gorm:"primaryKey;index:idx_bookmark_post"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	Post      postModel `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (bookmarkModel) TableName() string { return "bookmarks" }

type mediaModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	PostID     *int64    `gorm:"index:idx_media_post"`
	UploaderID string    `gorm:"size:64;not null;index:idx_media_uploader"`
	ObjectKey  string    `gorm:"size:255;not null"`
	MediaType  string    `gorm:"size:10;not null;check:chk_media_type,media_type IN ('image','video','audio')"`
	URL        string    `gorm:"size:500;not null"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
}

func (mediaModel) TableName() string { return "media" }

type hashtagModel struct {
	ID   int64  `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"size:100;not null;uniqueIndex"`
}

func (hashtagModel) TableName() string { return "hashtags" }

type pollModel struct {
	ID     int64     `gorm:"primaryKey;autoIncrement"`
	PostID int64     `gorm:"not null;uniqueIndex"`
	EndsAt time.Time `gorm:"not null"`
	Post   postModel `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (pollModel) TableName() string { return "polls" }

type pollOptionModel struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	PollID      int64     `gorm:"not null;index:idx_poll_option_poll"`
	OptionIndex int       `gorm:"not null"`
	Text        string    `gorm:"size:25;not null"`
	VotesCount  int       `gorm:"not null;default:0"`
	Poll        pollModel `gorm:"foreignKey:PollID;constraint:OnDelete:CASCADE"`
}

func (pollOptionModel) TableName() string { return "poll_options" }

type pollVoteModel struct {
	UserID      string    `gorm:"primaryKey;size:64"`
	PollID      int64     `gorm:"primaryKey;index:idx_poll_vote_poll"`
	OptionIndex int       `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
}

func (pollVoteModel) TableName() string { return "poll_votes" }

type postHashtagModel struct {
	PostID    int64        `gorm:"primaryKey"`
	HashtagID int64        `gorm:"primaryKey;index:idx_post_hashtag_hashtag"`
	Post      postModel    `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Hashtag   hashtagModel `gorm:"foreignKey:HashtagID;constraint:OnDelete:CASCADE"`
}

func (postHashtagModel) TableName() string { return "post_hashtags" }

type mentionInput struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type postMentionModel struct {
	PostID   int64     `gorm:"primaryKey"`
	UserID   string    `gorm:"primaryKey;size:64"`
	Username string    `gorm:"size:50;not null"`
	Post     postModel `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (postMentionModel) TableName() string { return "post_mentions" }

// reportModel matérialise le signalement d'un post par un utilisateur. L'index
// unique (reporter_id, post_id) empêche un même utilisateur de signaler deux
// fois le même post. Le statut suit le cycle de modération.
type reportModel struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	ReporterID  string    `gorm:"size:64;not null;uniqueIndex:idx_report_reporter_post"`
	PostID      int64     `gorm:"not null;uniqueIndex:idx_report_reporter_post;index:idx_report_post"`
	Reason      string    `gorm:"size:255;not null"`
	Status      string    `gorm:"size:20;not null;default:pending"`
	ModeratorID *string   `gorm:"size:64"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	// FK avec ON DELETE CASCADE : supprimer un post supprime ses signalements.
	Post postModel `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

func (reportModel) TableName() string { return "reports" }

const (
	reportStatusPending  = "pending"
	reportStatusResolved = "resolved"
	reportStatusRejected = "rejected"
	colModeratorID       = "moderator_id"
	colStatus            = "status"
)

// â"€â"€ DTOs â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€

type mediaResponse struct {
	ID        int64  `json:"id"`
	MediaType string `json:"media_type"`
	URL       string `json:"url"`
}

type postResponse struct {
	ID              int64             `json:"id"`
	Content         string            `json:"content"`
	AuthorID        string            `json:"author_id"`
	AuthorUsername  string            `json:"author_username"`
	AuthorAvatarURL *string           `json:"author_avatar_url,omitempty"`
	ParentID        *int64            `json:"parent_id"`
	QuoteID         *int64            `json:"quote_id,omitempty"`
	LikesCount      int               `json:"likes_count"`
	CommentsCount   int               `json:"comments_count"`
	RebreezeCount   int               `json:"rebreeze_count"`
	CreatedAt       string            `json:"created_at"`
	Media           []mediaResponse   `json:"media"`
	LikedByMe       bool              `json:"liked_by_me"`
	RebreezedByMe   bool              `json:"rebreezed_by_me"`
	BookmarkedByMe  bool              `json:"bookmarked_by_me"`
	Hashtags        []string          `json:"hashtags"`
	Mentions        []mentionResponse `json:"mentions"`
	Poll            *pollResponse     `json:"poll,omitempty"`
	QuotedPost      *postResponse     `json:"quoted_post,omitempty"`
}

type mentionResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type pollOptionResponse struct {
	ID          int64  `json:"id"`
	OptionIndex int    `json:"option_index"`
	Text        string `json:"text"`
	VotesCount  int    `json:"votes_count"`
}

type pollResponse struct {
	ID         int64                `json:"id"`
	EndsAt     string               `json:"ends_at"`
	TotalVotes int                  `json:"total_votes"`
	MyVote     *int                 `json:"my_vote"`
	Options    []pollOptionResponse `json:"options"`
}

func toPostResponse(p postModel) postResponse {
	return postResponse{
		ID:             p.ID,
		Content:        p.Content,
		AuthorID:       p.AuthorID,
		AuthorUsername: p.AuthorUsername,
		ParentID:       p.ParentID,
		QuoteID:        p.QuoteID,
		LikesCount:     p.LikesCount,
		CommentsCount:  p.CommentsCount,
		RebreezeCount:  p.RebreezeCount,
		CreatedAt:      p.CreatedAt.UTC().Format(time.RFC3339),
		Media:          []mediaResponse{},
		Hashtags:       []string{},
		Mentions:       []mentionResponse{},
	}
}

// â"€â"€ Helpers â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€

func postIDs(posts []postModel) []int64 {
	ids := make([]int64, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	return ids
}

func uniqueAuthorIDs(posts []postModel) []string {
	seen := make(map[string]struct{}, len(posts))
	ids := make([]string, 0, len(posts))
	for _, p := range posts {
		if _, ok := seen[p.AuthorID]; !ok {
			seen[p.AuthorID] = struct{}{}
			ids = append(ids, p.AuthorID)
		}
	}
	return ids
}

func fetchMediaByPostIDs(db *gorm.DB, ids []int64) map[int64][]mediaResponse {
	result := make(map[int64][]mediaResponse)
	if len(ids) == 0 {
		return result
	}
	var rows []mediaModel
	db.Where("post_id IN ?", ids).Find(&rows)
	for _, m := range rows {
		if m.PostID != nil {
			result[*m.PostID] = append(result[*m.PostID], mediaResponse{
				ID:        m.ID,
				MediaType: m.MediaType,
				URL:       m.URL,
			})
		}
	}
	return result
}

func fetchLikedByUser(db *gorm.DB, userID string, postIDs []int64) map[int64]bool {
	result := make(map[int64]bool)
	if userID == "" || len(postIDs) == 0 {
		return result
	}
	var likes []likeModel
	db.Where("user_id = ? AND post_id IN ?", userID, postIDs).Find(&likes)
	for _, l := range likes {
		result[l.PostID] = true
	}
	return result
}

func fetchRebreezedByUser(db *gorm.DB, userID string, postIDs []int64) map[int64]bool {
	result := make(map[int64]bool)
	if userID == "" || len(postIDs) == 0 {
		return result
	}
	var rows []rebreezersModel
	db.Where("user_id = ? AND post_id IN ?", userID, postIDs).Find(&rows)
	for _, r := range rows {
		result[r.PostID] = true
	}
	return result
}

func fetchBookmarkedByUser(db *gorm.DB, userID string, postIDs []int64) map[int64]bool {
	result := make(map[int64]bool)
	if userID == "" || len(postIDs) == 0 {
		return result
	}
	var rows []bookmarkModel
	db.Where("user_id = ? AND post_id IN ?", userID, postIDs).Find(&rows)
	for _, b := range rows {
		result[b.PostID] = true
	}
	return result
}

var hashtagRe = regexp.MustCompile(`#([\p{L}\p{N}_]{1,100})`)

func extractHashtags(content string) []string {
	matches := hashtagRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}

// colName est le nom de la colonne/paramètre « name » (hashtag), partagé entre
// l'upsert et le routing par hashtag.
const colName = "name"

func upsertHashtags(db *gorm.DB, postID int64, tags []string) {
	for _, tag := range tags {
		// Upsert concurrency-safe : sous charge concurrente (ex. seed), deux posts
		// partageant un nouveau hashtag courent sur la même insertion. ON CONFLICT
		// évite l'erreur de clé dupliquée qui, sinon, avorte la transaction du post
		// (500). Le DO UPDATE (no-op sur name) force le RETURNING id, indisponible
		// avec DO NOTHING.
		h := hashtagModel{Name: tag}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colName}},
			DoUpdates: clause.Assignments(map[string]any{colName: tag}),
		}).Create(&h).Error; err != nil {
			continue
		}
		ph := postHashtagModel{PostID: postID, HashtagID: h.ID}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&ph).Error; err != nil {
			continue
		}
	}
}

func fetchHashtagsByPostIDs(db *gorm.DB, ids []int64) map[int64][]string {
	result := make(map[int64][]string)
	if len(ids) == 0 {
		return result
	}
	type row struct {
		PostID int64
		Name   string
	}
	var rows []row
	db.Table("post_hashtags").
		Select("post_hashtags.post_id, hashtags.name").
		Joins("JOIN hashtags ON hashtags.id = post_hashtags.hashtag_id").
		Where("post_hashtags.post_id IN ?", ids).
		Scan(&rows)
	for _, r := range rows {
		result[r.PostID] = append(result[r.PostID], r.Name)
	}
	return result
}

func insertMentions(db *gorm.DB, postID int64, mentions []mentionInput) {
	for _, m := range mentions {
		if m.ID == "" || m.Username == "" {
			continue
		}
		pm := postMentionModel{PostID: postID, UserID: m.ID, Username: m.Username}
		db.Where(postMentionModel{PostID: postID, UserID: m.ID}).FirstOrCreate(&pm)
	}
}

func fetchPollsByPostIDs(db *gorm.DB, ids []int64, userID string) map[int64]*pollResponse {
	result := make(map[int64]*pollResponse)
	if len(ids) == 0 {
		return result
	}
	var polls []pollModel
	db.Where("post_id IN ?", ids).Find(&polls)
	if len(polls) == 0 {
		return result
	}
	pollIDs := make([]int64, len(polls))
	pollByID := make(map[int64]*pollModel)
	for i := range polls {
		pollIDs[i] = polls[i].ID
		pollByID[polls[i].ID] = &polls[i]
	}

	var options []pollOptionModel
	db.Where("poll_id IN ?", pollIDs).Order("option_index ASC").Find(&options)

	var myVotes []pollVoteModel
	if userID != "" {
		db.Where("poll_id IN ? AND user_id = ?", pollIDs, userID).Find(&myVotes)
	}
	myVoteMap := make(map[int64]int)
	for _, v := range myVotes {
		myVoteMap[v.PollID] = v.OptionIndex
	}

	optionsByPoll := make(map[int64][]pollOptionResponse)
	totalByPoll := make(map[int64]int)
	for _, o := range options {
		optionsByPoll[o.PollID] = append(optionsByPoll[o.PollID], pollOptionResponse{
			ID:          o.ID,
			OptionIndex: o.OptionIndex,
			Text:        o.Text,
			VotesCount:  o.VotesCount,
		})
		totalByPoll[o.PollID] += o.VotesCount
	}

	for _, p := range polls {
		pr := &pollResponse{
			ID:         p.ID,
			EndsAt:     p.EndsAt.UTC().Format(time.RFC3339),
			TotalVotes: totalByPoll[p.ID],
			Options:    optionsByPoll[p.ID],
		}
		if idx, voted := myVoteMap[p.ID]; voted {
			pr.MyVote = &idx
		}
		if pr.Options == nil {
			pr.Options = []pollOptionResponse{}
		}
		result[p.PostID] = pr
	}
	return result
}

func fetchMentionsByPostIDs(db *gorm.DB, ids []int64) map[int64][]mentionResponse {
	result := make(map[int64][]mentionResponse)
	if len(ids) == 0 {
		return result
	}
	type row struct {
		PostID   int64
		UserID   string
		Username string
	}
	var rows []row
	db.Table("post_mentions").
		Select("post_id, user_id, username").
		Where("post_id IN ?", ids).
		Scan(&rows)
	for _, r := range rows {
		result[r.PostID] = append(result[r.PostID], mentionResponse{ID: r.UserID, Username: r.Username})
	}
	return result
}

func enrichPosts(db *gorm.DB, posts []postModel, ids []int64, userID string, avatarMap ...map[string]*string) []postResponse {
	mediaMap := fetchMediaByPostIDs(db, ids)
	likedMap := fetchLikedByUser(db, userID, ids)
	rebreezedMap := fetchRebreezedByUser(db, userID, ids)
	bookmarkedMap := fetchBookmarkedByUser(db, userID, ids)
	hashtagMap := fetchHashtagsByPostIDs(db, ids)
	mentionMap := fetchMentionsByPostIDs(db, ids)
	pollMap := fetchPollsByPostIDs(db, ids, userID)
	var am map[string]*string
	if len(avatarMap) > 0 {
		am = avatarMap[0]
	}
	out := make([]postResponse, 0, len(posts))
	for _, p := range posts {
		r := toPostResponse(p)
		if m, ok := mediaMap[p.ID]; ok {
			r.Media = m
		}
		r.LikedByMe = likedMap[p.ID]
		r.RebreezedByMe = rebreezedMap[p.ID]
		r.BookmarkedByMe = bookmarkedMap[p.ID]
		if tags, ok := hashtagMap[p.ID]; ok {
			r.Hashtags = tags
		}
		if mentions, ok := mentionMap[p.ID]; ok {
			r.Mentions = mentions
		}
		if poll, ok := pollMap[p.ID]; ok {
			r.Poll = poll
		}
		if am != nil {
			r.AuthorAvatarURL = am[p.AuthorID]
		}
		out = append(out, r)
	}
	embedQuotedPosts(db, out, userID)
	return out
}

// embedQuotedPosts fetches and inlines the quoted post (one level) for each
// response that carries a QuoteID. Mutates out in place.
func embedQuotedPosts(db *gorm.DB, out []postResponse, userID string) {
	quoteIDs := make([]int64, 0)
	quoteIDSet := make(map[int64]bool)
	for _, r := range out {
		if r.QuoteID != nil && !quoteIDSet[*r.QuoteID] {
			quoteIDs = append(quoteIDs, *r.QuoteID)
			quoteIDSet[*r.QuoteID] = true
		}
	}
	if len(quoteIDs) == 0 {
		return
	}
	var quotedPosts []postModel
	db.Where("id IN ?", quoteIDs).Find(&quotedPosts)
	if len(quotedPosts) == 0 {
		return
	}
	qids := postIDs(quotedPosts)
	qMedia := fetchMediaByPostIDs(db, qids)
	qLiked := fetchLikedByUser(db, userID, qids)
	qRebreezed := fetchRebreezedByUser(db, userID, qids)
	qHashtags := fetchHashtagsByPostIDs(db, qids)
	qMentions := fetchMentionsByPostIDs(db, qids)
	qPolls := fetchPollsByPostIDs(db, qids, userID)
	enrichedQuotes := make([]postResponse, len(quotedPosts))
	for i, qp := range quotedPosts {
		qr := toPostResponse(qp)
		if m, ok := qMedia[qp.ID]; ok {
			qr.Media = m
		}
		qr.LikedByMe = qLiked[qp.ID]
		qr.RebreezedByMe = qRebreezed[qp.ID]
		if tags, ok := qHashtags[qp.ID]; ok {
			qr.Hashtags = tags
		}
		if mentions, ok := qMentions[qp.ID]; ok {
			qr.Mentions = mentions
		}
		if poll, ok := qPolls[qp.ID]; ok {
			qr.Poll = poll
		}
		enrichedQuotes[i] = qr
	}
	qIdx := make(map[int64]int, len(quotedPosts))
	for i, qp := range quotedPosts {
		qIdx[qp.ID] = i
	}
	for i := range out {
		if out[i].QuoteID != nil {
			if idx, ok := qIdx[*out[i].QuoteID]; ok {
				cp := enrichedQuotes[idx]
				out[i].QuotedPost = &cp
			}
		}
	}
}

// attachQuotedPost enrichit resp avec le post cité, sans récursion.
func (a *app) attachQuotedPost(resp *postResponse, quoteID int64, userID string) {
	var quoted postModel
	if err := a.db.First(&quoted, quoteID).Error; err != nil {
		return
	}
	qids := []int64{quoted.ID}
	qr := toPostResponse(quoted)
	if m := fetchMediaByPostIDs(a.db, qids)[quoted.ID]; m != nil {
		qr.Media = m
	}
	if tags := fetchHashtagsByPostIDs(a.db, qids)[quoted.ID]; tags != nil {
		qr.Hashtags = tags
	}
	if mentions := fetchMentionsByPostIDs(a.db, qids)[quoted.ID]; mentions != nil {
		qr.Mentions = mentions
	}
	if poll := fetchPollsByPostIDs(a.db, qids, userID)[quoted.ID]; poll != nil {
		qr.Poll = poll
	}
	resp.QuotedPost = &qr
}

func detectMediaType(contentType string) (string, bool) {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return mediaTypeImage, true
	case strings.HasPrefix(contentType, "video/"):
		return mediaTypeVideo, true
	case strings.HasPrefix(contentType, "audio/"):
		return mediaTypeAudio, true
	default:
		return "", false
	}
}

// extensionMediaInfo maps a lowercase file extension to its media type and a
// canonical MIME type. Used as a fallback when the browser sends an empty or
// generic Content-Type (e.g. application/octet-stream) for .mov, .mp3, etc.
var extensionMediaInfo = map[string]struct {
	mediaType   string
	contentType string
}{
	extJPG:  {mediaTypeImage, mimeJPEG},
	".jpeg": {mediaTypeImage, mimeJPEG},
	".png":  {mediaTypeImage, "image/png"},
	".gif":  {mediaTypeImage, "image/gif"},
	".webp": {mediaTypeImage, "image/webp"},
	".avif": {mediaTypeImage, "image/avif"},
	".bmp":  {mediaTypeImage, "image/bmp"},
	".heic": {mediaTypeImage, "image/heic"},
	".heif": {mediaTypeImage, "image/heif"},
	extMP4:  {mediaTypeVideo, mimeMP4},
	".m4v":  {mediaTypeVideo, mimeMP4},
	".mov":  {mediaTypeVideo, mimeQuicktime},
	extWebM: {mediaTypeVideo, "video/webm"},
	".avi":  {mediaTypeVideo, "video/x-msvideo"},
	".mkv":  {mediaTypeVideo, "video/x-matroska"},
	".mp3":  {mediaTypeAudio, mimeMPEG},
	".m4a":  {mediaTypeAudio, "audio/mp4"},
	".aac":  {mediaTypeAudio, "audio/aac"},
	".wav":  {mediaTypeAudio, "audio/wav"},
	".ogg":  {mediaTypeAudio, mimeOGG},
	".oga":  {mediaTypeAudio, mimeOGG},
	".flac": {mediaTypeAudio, "audio/flac"},
}

// detectMediaByExtension resolves the media type and a canonical Content-Type
// from a file extension. It lets uploads succeed when the browser/OS does not
// provide a usable MIME type for the chosen file.
func detectMediaByExtension(ext string) (mediaType, contentType string, ok bool) {
	info, found := extensionMediaInfo[strings.ToLower(ext)]
	if !found {
		return "", "", false
	}
	return info.mediaType, info.contentType, true
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func parsePostID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid post id")
		return 0, false
	}
	return id, true
}

func loadPost(db *gorm.DB, c *gin.Context, id int64) (postModel, bool) {
	var post postModel
	if err := db.First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "post not found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPost})
		}
		return postModel{}, false
	}
	return post, true
}

type maxFileReader struct {
	reader io.Reader
	read   int64
	max    int64
}

func (r *maxFileReader) Read(p []byte) (int, error) {
	if r.read >= r.max {
		var probe [1]byte
		n, err := r.reader.Read(probe[:])
		if n > 0 {
			return 0, errMediaTooLarge
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("probe media reader: %w", err)
		}
		if errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		return 0, nil
	}
	remaining := r.max - r.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("read media chunk: %w", err)
	}
	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}
	return n, nil
}

func normalizeMediaIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func validateContent(c *gin.Context, content string, hasMedia bool) bool {
	if strings.TrimSpace(content) == "" && !hasMedia {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "content or media is required")
		return false
	}
	if len(content) > 280 {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "content must not exceed 280 characters")
		return false
	}
	return true
}

type mediaUploadPart struct {
	part        *multipart.Part
	mediaType   string
	contentType string
	objectKey   string
}

func attachMediaToPost(tx *gorm.DB, mediaIDs []int64, uploaderID string, postID int64) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	result := tx.Model(&mediaModel{}).
		Where("id IN ? AND post_id IS NULL AND uploader_id = ?", mediaIDs, uploaderID).
		Update("post_id", postID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != int64(len(mediaIDs)) {
		return errInvalidMediaIDs
	}
	return nil
}

// â"€â"€ Config â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€

type minioCfg struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	PublicURL string
}

// â"€â"€ App â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€

type app struct {
	db          *gorm.DB
	mc          *minio.Client
	mcfg        minioCfg
	notifURL    string
	userURL     string
	internalKey string
}

var mentionContentRE = regexp.MustCompile(`@([a-zA-Z0-9_]{1,50})`)

// resolveMentions enrichit la liste de mentions explicites en parsant le contenu
// pour détecter les @username non sélectionnés via le dropdown d'autocomplétion.
func (a *app) resolveMentions(content string, explicit []mentionInput) []mentionInput {
	if a.userURL == "" {
		return explicit
	}
	known := make(map[string]bool, len(explicit))
	for _, m := range explicit {
		known[strings.ToLower(m.Username)] = true
	}
	var toResolve []string
	for _, match := range mentionContentRE.FindAllStringSubmatch(content, -1) {
		u := strings.ToLower(match[1])
		if !known[u] {
			known[u] = true
			toResolve = append(toResolve, u)
		}
	}
	if len(toResolve) == 0 {
		return explicit
	}
	data, _ := json.Marshal(map[string]any{"usernames": toResolve})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.userURL+"/internal/users/resolve-mentions", bytes.NewReader(data))
	if err != nil {
		return explicit
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", a.internalKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("resolveMentions: %v", err)
		return explicit
	}
	defer func() { _ = resp.Body.Close() }()
	var result struct {
		Data []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return explicit
	}
	merged := explicit
	for _, u := range result.Data {
		merged = append(merged, mentionInput{ID: u.ID, Username: u.Username})
	}
	return merged
}

// ── Notification helpers ──────────────────────────────────────────────────────

func (a *app) sendNotifAsync(payload map[string]string) {
	if a.notifURL == "" {
		return
	}
	go func() {
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.notifURL+"/internal/notifications", bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Key", a.internalKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("sendNotifAsync: %v", err)
			return
		}
		if resp.StatusCode >= 300 {
			log.Printf("sendNotifAsync: unexpected status %s", resp.Status)
		}
		_ = resp.Body.Close()
	}()
}

func (a *app) sendNewPostAsync(authorID, authorUsername, postID string) {
	if a.notifURL == "" {
		return
	}
	go func() {
		data, _ := json.Marshal(map[string]string{
			"author_id":       authorID,
			"author_username": authorUsername,
			"post_id":         postID,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.notifURL+"/internal/notifications/new-post", bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Key", a.internalKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("sendNewPostAsync: %v", err)
			return
		}
		if resp.StatusCode >= 300 {
			log.Printf("sendNewPostAsync: unexpected status %s", resp.Status)
		}
		_ = resp.Body.Close()
	}()
}

func (a *app) callerID(c *gin.Context) string {
	id, _ := shared.IdentityFromContext(c)
	return id.UserID
}

// canViewProfile interroge le user-service (endpoint interne) pour savoir si
// viewerID peut consulter les posts/re-posts de targetID. Fail-closed : toute
// erreur (réseau, statut ≠ 200) renvoie false afin de ne jamais fuiter le contenu
// d'un compte privé. Si USER_SERVICE_URL n'est pas configurée (dev/local), on
// laisse passer pour ne pas casser l'environnement de développement.
func (a *app) canViewProfile(viewerID, targetID string) bool {
	if a.userURL == "" {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	url := fmt.Sprintf("%s/internal/users/%s/can-view?viewer=%s",
		a.userURL, neturl.PathEscape(targetID), neturl.QueryEscape(viewerID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-Internal-Key", a.internalKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("canViewProfile: %v", err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var body struct {
		CanView bool `json:"canView"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false
	}
	return body.CanView
}

// fetchAuthorAvatars interroge le user-service pour les URLs d'avatar d'une liste
// d'auteurs. Renvoie une map authorID → *avatarURL. En cas d'erreur, renvoie nil
// (les avatars s'affichent alors sans photo, ce qui est acceptable).
func (a *app) fetchAuthorAvatars(authorIDs []string) map[string]*string {
	if a.userURL == "" || len(authorIDs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data, err := json.Marshal(map[string][]string{"ids": authorIDs})
	if err != nil {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.userURL+"/internal/users/batch-avatars", bytes.NewReader(data))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", a.internalKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("fetchAuthorAvatars: %v", err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var body struct {
		Data []struct {
			ID        string  `json:"id"`
			AvatarURL *string `json:"avatarUrl"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	result := make(map[string]*string, len(body.Data))
	for _, u := range body.Data {
		result[u.ID] = u.AvatarURL
	}
	return result
}

// hiddenAuthors interroge le user-service pour la liste des comptes privés que
// viewerID ne peut pas voir (non abonné). Ces auteurs sont exclus du feed global
// afin que leurs posts n'apparaissent pas aux non-abonnés. En cas d'erreur réseau
// ou si USER_SERVICE_URL n'est pas configurée (dev/local), renvoie nil : le feed
// reste fonctionnel (la protection stricte du contenu reste assurée par les
// endpoints /users/:id/posts et /rebreezed, eux fail-closed).
func (a *app) hiddenAuthors(viewerID string) []string {
	if a.userURL == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	url := fmt.Sprintf("%s/internal/users/hidden-authors?viewer=%s",
		a.userURL, neturl.QueryEscape(viewerID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("X-Internal-Key", a.internalKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("hiddenAuthors: %v", err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var body struct {
		Hidden []string `json:"hidden"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	return body.Hidden
}

// abortIfPrivate renvoie true (et répond 403) si l'appelant n'a pas le droit de
// voir le profil cible. Factorisé entre userPosts et userRebreezed.
func (a *app) abortIfPrivate(c *gin.Context, targetID string) bool {
	if a.canViewProfile(a.callerID(c), targetID) {
		return false
	}
	shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "this account is private")
	return true
}

func (a *app) globalFeed(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	callerID := a.callerID(c)
	// Exclut les posts des comptes privés que l'appelant ne suit pas, afin qu'ils
	// n'apparaissent pas dans le feed global.
	hidden := a.hiddenAuthors(callerID)

	base := a.db.Model(&postModel{}).Where("parent_id IS NULL")
	if len(hidden) > 0 {
		base = base.Where("author_id NOT IN ?", hidden)
	}

	var total int64
	base.Session(&gorm.Session{}).Count(&total)

	var posts []postModel
	if err := base.Session(&gorm.Session{}).Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPosts})
		return
	}
	ids := postIDs(posts)
	authorIDs := uniqueAuthorIDs(posts)
	avatarMap := a.fetchAuthorAvatars(authorIDs)
	c.JSON(http.StatusOK, gin.H{
		keyData:  enrichPosts(a.db, posts, ids, callerID, avatarMap),
		keyPage:  page,
		keyLimit: limit,
		keyTotal: total,
	})
}

// notifyMentions sends a mention notification to each mentioned user, skipping the actor and any IDs in skip.
func (a *app) notifyMentions(mentions []mentionInput, actorID, actorUsername string, entityID int64, skip ...string) {
	skipped := map[string]bool{actorID: true}
	for _, s := range skip {
		skipped[s] = true
	}
	for _, m := range mentions {
		if skipped[m.ID] {
			continue
		}
		a.sendNotifAsync(map[string]string{
			notifKeyUserID:     m.ID,
			notifKeyType:       notifTypeMention,
			notifKeyActorID:    actorID,
			notifKeyActorUser:  actorUsername,
			notifKeyEntityID:   strconv.FormatInt(entityID, 10),
			notifKeyEntityType: notifEntityPost,
		})
	}
}

// buildPostResponse assembles the API response for a freshly created post or reply.
func (a *app) buildPostResponse(post postModel, tags []string) postResponse {
	r := toPostResponse(post)
	r.Hashtags = tags
	r.Mentions = fetchMentionsByPostIDs(a.db, []int64{post.ID})[post.ID]
	if r.Mentions == nil {
		r.Mentions = []mentionResponse{}
	}
	var media []mediaModel
	a.db.Where("post_id = ?", post.ID).Find(&media)
	for _, m := range media {
		r.Media = append(r.Media, mediaResponse{ID: m.ID, MediaType: m.MediaType, URL: m.URL})
	}
	return r
}

type pollInput struct {
	Options      []string `json:"options"`
	DurationDays int      `json:"duration_days"`
}

// validatePollInput checks the poll payload (incl. mutual exclusion with media)
// and normalizes its duration. It aborts the request and returns false on error.
func validatePollInput(c *gin.Context, poll *pollInput, hasMedia bool) bool {
	if hasMedia {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot combine poll and media")
		return false
	}
	if len(poll.Options) < 2 || len(poll.Options) > 4 {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "poll must have 2 to 4 options")
		return false
	}
	for _, opt := range poll.Options {
		if strings.TrimSpace(opt) == "" || len(opt) > 25 {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "each poll option must be 1–25 characters")
			return false
		}
	}
	if poll.DurationDays < 1 || poll.DurationDays > 7 {
		poll.DurationDays = 1
	}
	return true
}

// attachPoll fills resp.Poll with the post's poll, if one exists.
func (a *app) attachPoll(resp *postResponse, postID int64, userID string) {
	if polls := fetchPollsByPostIDs(a.db, []int64{postID}, userID); polls[postID] != nil {
		resp.Poll = polls[postID]
	}
}

// createPollInTx persists a poll and its options inside an existing transaction.
func createPollInTx(tx *gorm.DB, postID int64, input *pollInput) error {
	poll := pollModel{
		PostID: postID,
		EndsAt: time.Now().UTC().Add(time.Duration(input.DurationDays) * 24 * time.Hour),
	}
	if err := tx.Create(&poll).Error; err != nil {
		return err
	}
	for i, opt := range input.Options {
		option := pollOptionModel{PollID: poll.ID, OptionIndex: i, Text: strings.TrimSpace(opt)}
		if err := tx.Create(&option).Error; err != nil {
			return err
		}
	}
	return nil
}

// validateNewPostInputs checks media count, poll, content and quoted-post
// existence. Writes the error response itself and returns false on failure.
func (a *app) validateNewPostInputs(c *gin.Context, content string, mediaIDs []int64, poll *pollInput, quoteID *int64) bool {
	if len(mediaIDs) > 4 {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "maximum 4 media per post")
		return false
	}
	hasMedia := len(mediaIDs) > 0
	if poll != nil && !validatePollInput(c, poll, hasMedia) {
		return false
	}
	if !validateContent(c, content, hasMedia || poll != nil || quoteID != nil) {
		return false
	}
	if quoteID != nil {
		var count int64
		if err := a.db.Model(&postModel{}).Where("id = ?", *quoteID).Count(&count).Error; err != nil || count == 0 {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "quoted post not found")
			return false
		}
	}
	return true
}

// persistPost inserts the post and its related data (media, hashtags, poll)
// inside a single transaction.
func (a *app) persistPost(post *postModel, mediaIDs []int64, tags []string, poll *pollInput) error {
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		if err := attachMediaToPost(tx, mediaIDs, post.AuthorID, post.ID); err != nil {
			return err
		}
		upsertHashtags(tx, post.ID, tags)
		if poll != nil {
			return createPollInTx(tx, post.ID, poll)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("persist post: %w", err)
	}
	return nil
}

func (a *app) createPost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)

	var body struct {
		Content        string         `json:"content"`
		AuthorUsername string         `json:"author_username"`
		MediaIDs       []int64        `json:"media_ids"`
		Mentions       []mentionInput `json:"mentions"`
		Poll           *pollInput     `json:"poll"`
		QuoteID        *int64         `json:"quote_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgInvalidBody)
		return
	}
	mediaIDs := normalizeMediaIDs(body.MediaIDs)
	content := strings.TrimSpace(body.Content)
	if !a.validateNewPostInputs(c, content, mediaIDs, body.Poll, body.QuoteID) {
		return
	}

	post := postModel{Content: content, AuthorID: identity.UserID, AuthorUsername: body.AuthorUsername, QuoteID: body.QuoteID}
	tags := extractHashtags(content)
	// Le thème n'est jamais fourni par l'auteur ni figé à la création : il émerge
	// des données (clusters de hashtags / comportement) et est posé a posteriori
	// par la passe d'inférence (cf. theme_infer.go).
	if err := a.persistPost(&post, mediaIDs, tags, body.Poll); err != nil {
		if errors.Is(err, errInvalidMediaIDs) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "one or more media ids are invalid")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to create post"})
		return
	}
	mentions := a.resolveMentions(content, body.Mentions)
	insertMentions(a.db, post.ID, mentions)
	a.notifyMentions(mentions, identity.UserID, body.AuthorUsername, post.ID)
	a.sendNewPostAsync(identity.UserID, body.AuthorUsername, strconv.FormatInt(post.ID, 10))
	// Rédiger un breeze renforce les préférences de l'auteur sur ses propres
	// hashtags et thème (la dimension auteur s'auto-ignore dans bumpAffinity).
	bumpAffinity(a.db, identity.UserID, post, tags, weightAuthored)
	resp := a.buildPostResponse(post, tags)
	a.attachPoll(&resp, post.ID, identity.UserID)
	if post.QuoteID != nil {
		a.attachQuotedPost(&resp, *post.QuoteID, identity.UserID)
	}
	c.JSON(http.StatusCreated, gin.H{keyData: resp})
}

func (a *app) votePoll(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	post, ok := loadPost(a.db, c, postID)
	if !ok {
		return
	}
	if post.AuthorID == identity.UserID {
		shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "cannot vote on own poll")
		return
	}

	var body struct {
		OptionIndex int `json:"option_index"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgInvalidBody)
		return
	}

	var poll pollModel
	if err := a.db.Where("post_id = ?", postID).First(&poll).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "poll not found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to fetch poll"})
		}
		return
	}
	if time.Now().UTC().After(poll.EndsAt) {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "poll has ended")
		return
	}

	var existing pollVoteModel
	if err := a.db.Where("poll_id = ? AND user_id = ?", poll.ID, identity.UserID).First(&existing).Error; err == nil {
		shared.AbortError(c, http.StatusConflict, shared.ErrConflict, "already voted")
		return
	}

	var option pollOptionModel
	if err := a.db.Where("poll_id = ? AND option_index = ?", poll.ID, body.OptionIndex).First(&option).Error; err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid option index")
		return
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		vote := pollVoteModel{UserID: identity.UserID, PollID: poll.ID, OptionIndex: body.OptionIndex}
		if err := tx.Create(&vote).Error; err != nil {
			return err
		}
		return tx.Model(&pollOptionModel{}).Where("id = ?", option.ID).
			UpdateColumn("votes_count", gorm.Expr("votes_count + 1")).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to register vote"})
		return
	}

	polls := fetchPollsByPostIDs(a.db, []int64{postID}, identity.UserID)
	c.JSON(http.StatusOK, gin.H{keyData: polls[postID]})
}

func (a *app) deletePost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	id, ok := parsePostID(c)
	if !ok {
		return
	}
	post, ok := loadPost(a.db, c, id)
	if !ok {
		return
	}
	// Le staff (modérateur/administrateur) peut supprimer n'importe quel post ;
	// un utilisateur ordinaire uniquement le sien.
	if post.AuthorID != identity.UserID && !shared.IsStaff(identity.Role) {
		shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "cannot delete someone else's post")
		return
	}

	if err := a.purgePost(post); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to delete post"})
		return
	}
	c.Status(http.StatusNoContent)
}

// purgePost supprime un post : décrémente le compteur de réponses du parent le
// cas échéant, supprime la ligne (cascade sur likes/médias/hashtags/mentions/
// signalements) puis nettoie les objets média dans MinIO en arrière-plan.
func (a *app) purgePost(post postModel) error {
	var mediaRows []mediaModel
	a.db.Where("post_id = ?", post.ID).Find(&mediaRows)

	if post.ParentID != nil {
		a.db.Model(&postModel{}).Where("id = ?", *post.ParentID).
			UpdateColumn("comments_count", gorm.Expr("GREATEST(comments_count - 1, 0)"))
	}
	if err := a.db.Delete(&postModel{}, post.ID).Error; err != nil {
		return err
	}
	for _, m := range mediaRows {
		go func(key string) {
			_ = a.mc.RemoveObject(context.Background(), a.mcfg.Bucket, key, minio.RemoveObjectOptions{})
		}(m.ObjectKey)
	}
	return nil
}

func (a *app) searchPosts(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "q is required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	pattern := "%" + q + "%"
	var total int64
	a.db.Model(&postModel{}).Where("parent_id IS NULL AND content ILIKE ?", pattern).Count(&total)

	var posts []postModel
	if err := a.db.Where("parent_id IS NULL AND content ILIKE ?", pattern).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPosts})
		return
	}

	ids := postIDs(posts)
	c.JSON(http.StatusOK, gin.H{
		keyData:  enrichPosts(a.db, posts, ids, a.callerID(c)),
		keyPage:  page,
		keyLimit: limit,
		keyTotal: total,
	})
}

func (a *app) getPost(c *gin.Context) {
	id, ok := parsePostID(c)
	if !ok {
		return
	}
	post, ok := loadPost(a.db, c, id)
	if !ok {
		return
	}
	ids := []int64{post.ID}
	avatarMap := a.fetchAuthorAvatars([]string{post.AuthorID})
	resp := enrichPosts(a.db, []postModel{post}, ids, a.callerID(c), avatarMap)
	c.JSON(http.StatusOK, gin.H{keyData: resp[0]})
}

func (a *app) userPosts(c *gin.Context) {
	authorID := c.Param("id")
	if a.abortIfPrivate(c, authorID) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Onglet "Posts" du profil : posts de premier niveau ET réponses, mêlés
	// par ordre antéchronologique (à la Twitter/X « Posts & replies »).
	var total int64
	a.db.Model(&postModel{}).Where("author_id = ?", authorID).Count(&total)

	var posts []postModel
	if err := a.db.Where("author_id = ?", authorID).Order("created_at DESC").Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPosts})
		return
	}
	ids := postIDs(posts)
	avatarMap := a.fetchAuthorAvatars([]string{authorID})
	c.JSON(http.StatusOK, gin.H{
		keyData:  enrichPosts(a.db, posts, ids, a.callerID(c), avatarMap),
		keyPage:  page,
		keyLimit: limit,
		keyTotal: total,
	})
}

func buildMediaUploadPart(part *multipart.Part) (mediaUploadPart, error) {
	if part.FileName() == "" {
		return mediaUploadPart{}, errMediaFileRequired
	}

	contentType := part.Header.Get("Content-Type")
	ext := filepath.Ext(part.FileName())

	mediaType, ok := detectMediaType(contentType)
	if !ok {
		// The browser/OS may not provide a usable MIME type (e.g. empty or
		// application/octet-stream for .mov/.mp3). Fall back to the extension
		// and adopt its canonical Content-Type so playback works afterwards.
		var resolvedType string
		mediaType, resolvedType, ok = detectMediaByExtension(ext)
		if !ok {
			return mediaUploadPart{}, errUnsupportedMedia
		}
		contentType = resolvedType
	}

	if contentType == "" {
		contentType = mimeOctetStream
	}

	if ext == "" {
		fallback := map[string]string{mediaTypeImage: ".jpg", mediaTypeVideo: ".mp4", mediaTypeAudio: ".webm"}
		ext = fallback[mediaType]
	}

	return mediaUploadPart{
		part:        part,
		mediaType:   mediaType,
		contentType: contentType,
		objectKey:   fmt.Sprintf("media/%s%s", randomHex(16), ext),
	}, nil
}

// engagementOps holds the string parameters that differ between like/rebreeze operations.
type engagementOps struct {
	conflictMsg  string
	checkErrMsg  string
	createErrMsg string
	notFoundMsg  string
	deleteErrMsg string
	notifType    string
	successMsg   string
	countCol     string
	// affinityWeight est le poids ajouté (resp. retiré) aux préférences du lecteur
	// lors de l'engagement (resp. désengagement). 0 = pas de mise à jour.
	affinityWeight float64
}

// addEngagement encapsulates the check-duplicate → create → increment-count → notify flow.
func (a *app) addEngagement(
	c *gin.Context,
	identity shared.Identity,
	post postModel,
	postID int64,
	existing interface{},
	record interface{},
	ops engagementOps,
) {
	err := a.db.Where("user_id = ? AND post_id = ?", identity.UserID, postID).First(existing).Error
	if err == nil {
		shared.AbortError(c, http.StatusConflict, shared.ErrConflict, ops.conflictMsg)
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: ops.checkErrMsg})
		return
	}
	if err := a.db.Create(record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: ops.createErrMsg})
		return
	}
	a.db.Model(&post).UpdateColumn(ops.countCol, gorm.Expr(ops.countCol+" + 1"))
	if ops.affinityWeight != 0 {
		a.bumpAffinityForPost(identity.UserID, post, ops.affinityWeight)
	}
	a.sendNotifAsync(map[string]string{
		notifKeyUserID:     post.AuthorID,
		notifKeyType:       ops.notifType,
		notifKeyActorID:    identity.UserID,
		notifKeyActorUser:  identity.Username,
		notifKeyEntityID:   strconv.FormatInt(postID, 10),
		notifKeyEntityType: notifEntityPost,
	})
	c.JSON(http.StatusCreated, gin.H{keyMessage: ops.successMsg})
}

// removeEngagement encapsulates the delete-record → decrement-count flow.
func (a *app) removeEngagement(
	c *gin.Context,
	identity shared.Identity,
	postID int64,
	record interface{},
	ops engagementOps,
) {
	result := a.db.Where("user_id = ? AND post_id = ?", identity.UserID, postID).Delete(record)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: ops.deleteErrMsg})
		return
	}
	if result.RowsAffected == 0 {
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, ops.notFoundMsg)
		return
	}
	a.db.Model(&postModel{}).Where("id = ?", postID).
		UpdateColumn(ops.countCol, gorm.Expr("GREATEST("+ops.countCol+" - 1, 0)"))
	if ops.affinityWeight != 0 {
		var post postModel
		if err := a.db.First(&post, postID).Error; err == nil {
			a.bumpAffinityForPost(identity.UserID, post, -ops.affinityWeight)
		}
	}
	c.Status(http.StatusNoContent)
}

func (a *app) likePost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	post, ok := loadPost(a.db, c, postID)
	if !ok {
		return
	}
	a.addEngagement(c, identity, post, postID, &likeModel{}, &likeModel{UserID: identity.UserID, PostID: postID}, engagementOps{
		conflictMsg: "post already liked", checkErrMsg: "failed to check existing like",
		createErrMsg: "failed to like post", notifType: notifTypeLike,
		successMsg: "post liked", countCol: colLikesCount, affinityWeight: weightLike,
	})
}

func (a *app) unlikePost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	a.removeEngagement(c, identity, postID, &likeModel{}, engagementOps{
		notFoundMsg: "like not found", deleteErrMsg: "failed to unlike post", countCol: colLikesCount,
		affinityWeight: weightLike,
	})
}

func (a *app) createReply(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	parentID, ok := parsePostID(c)
	if !ok {
		return
	}
	parent, ok := loadPost(a.db, c, parentID)
	if !ok {
		return
	}

	var body struct {
		Content        string         `json:"content"`
		AuthorUsername string         `json:"author_username"`
		MediaIDs       []int64        `json:"media_ids"`
		Mentions       []mentionInput `json:"mentions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgInvalidBody)
		return
	}
	mediaIDs := normalizeMediaIDs(body.MediaIDs)
	content := strings.TrimSpace(body.Content)
	if len(mediaIDs) > 4 {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "maximum 4 media per post")
		return
	}
	if !validateContent(c, content, len(mediaIDs) > 0) {
		return
	}

	reply := postModel{Content: content, AuthorID: identity.UserID, AuthorUsername: body.AuthorUsername, ParentID: &parentID}
	replyTags := extractHashtags(content)
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&reply).Error; err != nil {
			return err
		}
		if err := attachMediaToPost(tx, mediaIDs, identity.UserID, reply.ID); err != nil {
			return err
		}
		upsertHashtags(tx, reply.ID, replyTags)
		if err := tx.Model(&parent).UpdateColumn("comments_count", gorm.Expr("comments_count + 1")).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		if errors.Is(err, errInvalidMediaIDs) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "one or more media ids are invalid")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to create reply"})
		return
	}
	mentions := a.resolveMentions(content, body.Mentions)
	insertMentions(a.db, reply.ID, mentions)
	a.sendNotifAsync(map[string]string{
		notifKeyUserID:     parent.AuthorID,
		notifKeyType:       "comment",
		notifKeyActorID:    identity.UserID,
		notifKeyActorUser:  body.AuthorUsername,
		notifKeyEntityID:   strconv.FormatInt(parentID, 10),
		notifKeyEntityType: notifEntityPost,
	})
	a.notifyMentions(mentions, identity.UserID, body.AuthorUsername, reply.ID, parent.AuthorID)
	// Commenter signale un intérêt pour le contenu du post parent (ses hashtags,
	// son auteur, son thème) → renforce les préférences du commentateur.
	a.bumpAffinityForPost(identity.UserID, parent, weightComment)
	c.JSON(http.StatusCreated, gin.H{keyData: a.buildPostResponse(reply, replyTags)})
}

func (a *app) listReplies(c *gin.Context) {
	parentID, ok := parsePostID(c)
	if !ok {
		return
	}
	var replies []postModel
	if err := a.db.Where("parent_id = ?", parentID).Order("created_at ASC").Find(&replies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to fetch replies"})
		return
	}
	ids := postIDs(replies)
	c.JSON(http.StatusOK, gin.H{keyData: enrichPosts(a.db, replies, ids, a.callerID(c))})
}

func (a *app) postsByHashtag(c *gin.Context) {
	name := strings.ToLower(c.Param(colName))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	a.db.Table("posts").
		Joins("JOIN post_hashtags ON post_hashtags.post_id = posts.id").
		Joins("JOIN hashtags ON hashtags.id = post_hashtags.hashtag_id").
		Where("hashtags.name = ? AND posts.parent_id IS NULL", name).
		Count(&total)

	var posts []postModel
	if err := a.db.
		Joins("JOIN post_hashtags ON post_hashtags.post_id = posts.id").
		Joins("JOIN hashtags ON hashtags.id = post_hashtags.hashtag_id").
		Where("hashtags.name = ? AND posts.parent_id IS NULL", name).
		Order("posts.created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPosts})
		return
	}
	ids := postIDs(posts)
	c.JSON(http.StatusOK, gin.H{
		keyData:  enrichPosts(a.db, posts, ids, a.callerID(c)),
		keyPage:  page,
		keyLimit: limit,
		keyTotal: total,
	})
}

type trendingHashtag struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

func (a *app) trendingHashtags(c *gin.Context) {
	since := time.Now().UTC().Add(-24 * time.Hour)
	callerID := a.callerID(c)
	var results []trendingHashtag
	// Tendances par thème : on regroupe en tête les hashtags dont le thème
	// intéresse le lecteur (affinité-thème, via user_affinity), puis on classe par
	// volume 24h à l'intérieur de chaque niveau d'affinité. Le `count` renvoyé
	// reste le comptage brut (affiché tel quel) ; seul l'ORDRE est personnalisé.
	// Lecteur anonyme ou sans affinité → MAX(affinité)=0 partout → tri par volume
	// = tendances globales.
	a.db.Table("post_hashtags").
		Select("hashtags.name AS name, COUNT(*) AS count").
		Joins("JOIN hashtags ON hashtags.id = post_hashtags.hashtag_id").
		Joins("JOIN posts ON posts.id = post_hashtags.post_id").
		Joins("LEFT JOIN user_affinity ua ON ua.user_id = ? AND ua.kind = ? AND ua.ref = posts.theme", callerID, affinityKindTheme).
		Where("posts.created_at > ? AND posts.parent_id IS NULL", since).
		Group("hashtags.name").
		Order("MAX(COALESCE(ua.weight, 0)) DESC").
		Order("count DESC").
		Limit(10).
		Scan(&results)
	c.JSON(http.StatusOK, gin.H{keyData: results})
}

// postsByEngagement liste, du plus récent au plus ancien, les posts qu'un
// utilisateur a engagés via une table de liaison (post_id, user_id, created_at) :
// rebreezers pour l'onglet « Rebreezes », likes pour l'onglet « J'aime ».
func (a *app) postsByEngagement(c *gin.Context, joinTable string) {
	userID := c.Param("id")
	if a.abortIfPrivate(c, userID) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	a.db.Table(joinTable).Where("user_id = ?", userID).Count(&total)

	var posts []postModel
	if err := a.db.
		Joins(fmt.Sprintf("JOIN %[1]s ON %[1]s.post_id = posts.id", joinTable)).
		Where(joinTable+".user_id = ?", userID).
		Order(joinTable + ".created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPosts})
		return
	}
	ids := postIDs(posts)
	c.JSON(http.StatusOK, gin.H{
		keyData:  enrichPosts(a.db, posts, ids, a.callerID(c)),
		keyPage:  page,
		keyLimit: limit,
		keyTotal: total,
	})
}

func (a *app) userRebreezed(c *gin.Context) { a.postsByEngagement(c, "rebreezers") }

func (a *app) userLiked(c *gin.Context) { a.postsByEngagement(c, "likes") }

func (a *app) rebreezePost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	post, ok := loadPost(a.db, c, postID)
	if !ok {
		return
	}
	a.addEngagement(c, identity, post, postID, &rebreezersModel{}, &rebreezersModel{UserID: identity.UserID, PostID: postID}, engagementOps{
		conflictMsg: "post already rebreezed", checkErrMsg: "failed to check existing rebreeze",
		createErrMsg: "failed to rebreeze post", notifType: notifTypeRebreeze,
		successMsg: "post rebreezed", countCol: colRebreezeCount, affinityWeight: weightRebreeze,
	})
}

func (a *app) unrebreezePost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	a.removeEngagement(c, identity, postID, &rebreezersModel{}, engagementOps{
		notFoundMsg: "rebreeze not found", deleteErrMsg: "failed to unrebreeze post", countCol: colRebreezeCount,
		affinityWeight: weightRebreeze,
	})
}

// â"€â"€ Bookmarks (signets) â"€â"€

// userBookmarks renvoie les posts mis en signet par l'utilisateur courant. Les
// signets sont privés : seul leur propriétaire peut consulter sa liste.
func (a *app) userBookmarks(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	if c.Param("id") != identity.UserID {
		shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "bookmarks are private")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	a.db.Model(&bookmarkModel{}).Where("user_id = ?", identity.UserID).Count(&total)

	var posts []postModel
	if err := a.db.
		Joins("JOIN bookmarks ON bookmarks.post_id = posts.id").
		Where("bookmarks.user_id = ?", identity.UserID).
		Order("bookmarks.created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPosts})
		return
	}
	ids := postIDs(posts)
	c.JSON(http.StatusOK, gin.H{
		keyData:  enrichPosts(a.db, posts, ids, identity.UserID),
		keyPage:  page,
		keyLimit: limit,
		keyTotal: total,
	})
}

// bookmarkPost ajoute un post aux signets de l'utilisateur courant. Un post
// déjà en signet renvoie 409.
func (a *app) bookmarkPost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	post, ok := loadPost(a.db, c, postID)
	if !ok {
		return
	}
	var existing bookmarkModel
	err := a.db.Where("user_id = ? AND post_id = ?", identity.UserID, postID).First(&existing).Error
	if err == nil {
		shared.AbortError(c, http.StatusConflict, shared.ErrConflict, "post already bookmarked")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to check existing bookmark"})
		return
	}
	if err := a.db.Create(&bookmarkModel{UserID: identity.UserID, PostID: postID}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to bookmark post"})
		return
	}
	a.bumpAffinityForPost(identity.UserID, post, weightBookmark)
	c.JSON(http.StatusCreated, gin.H{keyMessage: "post bookmarked"})
}

// unbookmarkPost retire un post des signets de l'utilisateur courant.
func (a *app) unbookmarkPost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	result := a.db.Where("user_id = ? AND post_id = ?", identity.UserID, postID).Delete(&bookmarkModel{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to unbookmark post"})
		return
	}
	if result.RowsAffected == 0 {
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "bookmark not found")
		return
	}
	var post postModel
	if err := a.db.First(&post, postID).Error; err == nil {
		a.bumpAffinityForPost(identity.UserID, post, -weightBookmark)
	}
	c.Status(http.StatusNoContent)
}

// â"€â"€ Reports (signalement) â"€â"€

type reportResponse struct {
	ID                 int64   `json:"id"`
	PostID             int64   `json:"postId"`
	ReporterID         string  `json:"reporterId"`
	Reason             string  `json:"reason"`
	Status             string  `json:"status"`
	ModeratorID        *string `json:"moderatorId"`
	CreatedAt          string  `json:"createdAt"`
	PostContent        *string `json:"postContent"`
	PostAuthorID       *string `json:"postAuthorId"`
	PostAuthorUsername *string `json:"postAuthorUsername"`
}

// reportPost enregistre le signalement d'un post par l'utilisateur courant.
func (a *app) reportPost(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	postID, ok := parsePostID(c)
	if !ok {
		return
	}
	post, ok := loadPost(a.db, c, postID)
	if !ok {
		return
	}
	if post.AuthorID == identity.UserID {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot report your own post")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgInvalidBody)
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "reason is required")
		return
	}
	if len(reason) > 255 {
		reason = reason[:255]
	}

	report := reportModel{ReporterID: identity.UserID, PostID: postID, Reason: reason, Status: reportStatusPending}
	res := a.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&report)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to create report"})
		return
	}
	if res.RowsAffected == 0 {
		shared.AbortError(c, http.StatusConflict, shared.ErrConflict, "post already reported")
		return
	}
	c.JSON(http.StatusCreated, gin.H{keyMessage: "post reported"})
}

// listReports renvoie les signalements (staff uniquement), du plus récent au plus
// ancien, enrichis du contenu du post signalé. Filtrable par ?status=.
func (a *app) listReports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	base := a.db.Model(&reportModel{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		base = base.Where("status = ?", status)
	}

	var total int64
	base.Session(&gorm.Session{}).Count(&total)

	var reports []reportModel
	if err := base.Session(&gorm.Session{}).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to fetch reports"})
		return
	}

	// Charge les posts associés en une requête pour enrichir la réponse.
	ids := make([]int64, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.PostID)
	}
	postMap := make(map[int64]postModel)
	if len(ids) > 0 {
		var posts []postModel
		a.db.Where("id IN ?", ids).Find(&posts)
		for _, p := range posts {
			postMap[p.ID] = p
		}
	}

	out := make([]reportResponse, 0, len(reports))
	for _, r := range reports {
		resp := reportResponse{
			ID:          r.ID,
			PostID:      r.PostID,
			ReporterID:  r.ReporterID,
			Reason:      r.Reason,
			Status:      r.Status,
			ModeratorID: r.ModeratorID,
			CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		}
		if p, ok := postMap[r.PostID]; ok {
			content, authorID, username := p.Content, p.AuthorID, p.AuthorUsername
			resp.PostContent = &content
			resp.PostAuthorID = &authorID
			resp.PostAuthorUsername = &username
		}
		out = append(out, resp)
	}
	c.JSON(http.StatusOK, gin.H{keyData: out, keyPage: page, keyLimit: limit, keyTotal: total})
}

// updateReport traite un signalement (staff). action ∈ {resolve, reject, delete} :
//   - reject  : signalement infondé, post conservé.
//   - resolve : signalement traité, post conservé.
//   - delete  : supprime le post signalé (cascade sur ses signalements).
func (a *app) updateReport(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid report id")
		return
	}

	var body struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgInvalidBody)
		return
	}

	var report reportModel
	if err := a.db.First(&report, reportID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "report not found")
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to fetch report"})
		return
	}

	switch body.Action {
	case "reject":
		a.db.Model(&reportModel{}).Where("id = ?", report.ID).
			Updates(map[string]any{colStatus: reportStatusRejected, colModeratorID: identity.UserID})
	case "resolve":
		a.db.Model(&reportModel{}).Where("id = ?", report.ID).
			Updates(map[string]any{colStatus: reportStatusResolved, colModeratorID: identity.UserID})
	case "delete":
		post, ok := loadPost(a.db, c, report.PostID)
		if !ok {
			return
		}
		if err := a.purgePost(post); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to delete reported post"})
			return
		}
		// La suppression du post cascade sur ses signalements ; on note la
		// décision sur les autres signalements pendants du même post (déjà
		// supprimés par cascade, donc no-op si la FK a joué).
	default:
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "action must be resolve, reject or delete")
		return
	}
	c.JSON(http.StatusOK, gin.H{keyMessage: "report processed"})
}

// â"€â"€ main â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€â"€

// newRouter assemble les routes du post-service. Extrait de main pour être
// instanciable depuis les tests (cf. newUserRouter / newAuthRouter).
func newRouter(a *app) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.HandleMethodNotAllowed = true
	router.MaxMultipartMemory = maxMultipartRAM

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, shared.NewHealthResponse("post-service"))
	})
	router.GET("/posts", a.globalFeed)
	router.GET("/posts/for-you", a.forYouFeed)
	router.POST("/posts", shared.RequireIdentity(), a.createPost)
	router.GET("/posts/search", a.searchPosts)
	router.DELETE("/posts/:id", shared.RequireIdentity(), a.deletePost)
	router.GET("/posts/:id", a.getPost)
	router.GET("/users/:id/posts", a.userPosts)
	router.GET("/users/:id/rebreezed", a.userRebreezed)
	router.GET("/users/:id/liked", a.userLiked)
	router.GET("/users/:id/bookmarks", shared.RequireIdentity(), a.userBookmarks)
	router.POST("/posts/:id/bookmarks", shared.RequireIdentity(), a.bookmarkPost)
	router.DELETE("/posts/:id/bookmarks", shared.RequireIdentity(), a.unbookmarkPost)
	router.POST("/media/upload", shared.RequireIdentity(), a.uploadMedia)
	router.POST("/posts/:id/likes", shared.RequireIdentity(), a.likePost)
	router.DELETE("/posts/:id/likes", shared.RequireIdentity(), a.unlikePost)
	router.POST("/posts/:id/rebreezers", shared.RequireIdentity(), a.rebreezePost)
	router.DELETE("/posts/:id/rebreezers", shared.RequireIdentity(), a.unrebreezePost)
	router.POST("/posts/:id/poll/vote", shared.RequireIdentity(), a.votePoll)
	router.POST("/posts/:id/replies", shared.RequireIdentity(), a.createReply)
	router.GET("/posts/:id/replies", a.listReplies)
	router.GET("/hashtags/trending", a.trendingHashtags)
	router.GET("/hashtags/:name/posts", a.postsByHashtag)
	// Signalement : ouvert à tout utilisateur authentifié.
	router.POST("/posts/:id/report", shared.RequireIdentity(), a.reportPost)
	// Modération : réservé au staff (modérateur / administrateur).
	router.GET("/reports", shared.RequireRole(shared.RoleModerator, shared.RoleAdministrator), a.listReports)
	router.PATCH("/reports/:id", shared.RequireRole(shared.RoleModerator, shared.RoleAdministrator), a.updateReport)

	// Routes internes service-à-service (non exposées via le gateway).
	internal := router.Group("/internal", requireInternalKey(a.internalKey))
	internal.GET("/posts/suggested-authors", a.suggestedAuthors)
	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "method not allowed")
	})

	return router
}
