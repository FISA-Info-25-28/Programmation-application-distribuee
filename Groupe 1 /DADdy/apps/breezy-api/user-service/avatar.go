package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/shared"
)

// maxAvatarSize plafonne la taille d'un avatar uploadé (5 Mio) et maxBannerSize
// celle d'une bannière (10 Mio, car plus large à l'écran).
const (
	maxAvatarSize = 5 << 20
	maxBannerSize = 10 << 20
)

// allowedImageTypes liste les types MIME image acceptés (avatar et bannière),
// associés à l'extension de secours utilisée si le nom de fichier d'origine
// n'en porte pas.
var allowedImageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// newMinioClient se connecte à MinIO et garantit l'existence du bucket.
func newMinioClient(cfg minioConfig) (*minio.Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("connect minio: %w", err)
	}
	if err := ensureBucket(mc, cfg.Bucket); err != nil {
		return nil, err
	}
	return mc, nil
}

// ensureBucket creates the bucket if it doesn't exist and always (re)applies
// the public-read policy on objects — including on a pre-existing bucket —
// so that avatars remain directly servable to the browser via their URL.
func ensureBucket(mc *minio.Client, bucket string) error {
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket existence: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucket)
	if err := mc.SetBucketPolicy(ctx, bucket, policy); err != nil {
		return fmt.Errorf("set bucket policy: %w", err)
	}
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// imageUpload décrit un type d'image personnalisable du profil (avatar ou
// bannière) : sa taille max, le préfixe de clé MinIO et le libellé utilisé dans
// les messages d'erreur, plus les fonctions d'écriture en base et de réponse propres
// à chaque type.
type imageUpload struct {
	maxSize   int64
	keyPrefix string
	label     string
	// persist écrit la nouvelle URL/clé en base et renvoie l'ancienne clé d'objet
	// (nil si aucune), pour permettre la suppression de l'ancien fichier.
	persist func(userID, objectKey, url string) (*string, error)
	// respond émet la réponse HTTP de succès (200) avec l'URL publique.
	respond func(c *gin.Context, url string)
}

// handleImageUpload reçoit un fichier image (champ multipart "file"), le stocke
// dans MinIO sous <keyPrefix>/<userId>/<hash><ext>, met à jour le profil et
// supprime l'ancien fichier le cas échéant. Factorisé entre avatar et bannière.
func (s *userService) handleImageUpload(c *gin.Context, up imageUpload) {
	identity, _ := shared.IdentityFromContext(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "file field required")
		return
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Printf("failed to close uploaded %s: %v", up.label, cerr)
		}
	}()

	sizeMsg := fmt.Sprintf("%s must not exceed %dMB", up.label, up.maxSize>>20)
	if header.Size > up.maxSize {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, sizeMsg)
		return
	}

	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedImageTypes[strings.ToLower(contentType)]
	if !ok {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, fmt.Sprintf("%s must be a PNG, JPEG, WebP or GIF image", up.label))
		return
	}
	if e := strings.ToLower(filepath.Ext(header.Filename)); e != "" {
		ext = e
	}

	data, err := io.ReadAll(io.LimitReader(file, up.maxSize+1))
	if err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to read file")
		return
	}
	if int64(len(data)) > up.maxSize {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, sizeMsg)
		return
	}

	cfg := s.cfg.Minio
	objectKey := fmt.Sprintf("%s/%s/%s%s", up.keyPrefix, identity.UserID, randomHex(16), ext)
	if _, err := s.mc.PutObject(c.Request.Context(), cfg.Bucket, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		log.Printf("minio put %s error: %v", up.label, err)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, fmt.Sprintf("failed to store %s", up.label))
		return
	}

	url := fmt.Sprintf("%s/%s/%s", cfg.PublicURL, cfg.Bucket, objectKey)
	oldKey, err := up.persist(identity.UserID, objectKey, url)
	if err != nil {
		// Le profil n'a pas pu être mis à jour : on retire l'objet tout juste poussé
		// pour ne pas laisser de fichier orphelin.
		s.removeObject(objectKey)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, fmt.Sprintf("failed to update %s", up.label))
		return
	}

	if oldKey != nil && *oldKey != "" && *oldKey != objectKey {
		s.removeObject(*oldKey)
	}

	up.respond(c, url)
}

// handleImageDelete supprime l'image courante (objet MinIO + champs en base) via
// la fonction clear fournie. Factorisé entre avatar et bannière.
func (s *userService) handleImageDelete(c *gin.Context, clear func(userID string) (*string, error), label string) {
	identity, _ := shared.IdentityFromContext(c)

	oldKey, err := clear(identity.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, fmt.Sprintf("failed to delete %s", label))
		return
	}
	if oldKey != nil && *oldKey != "" {
		s.removeObject(*oldKey)
	}
	c.Status(http.StatusNoContent)
}

// uploadAvatarHandler stocke un nouvel avatar (PUT /users/me/avatar).
func (s *userService) uploadAvatarHandler(c *gin.Context) {
	s.handleImageUpload(c, imageUpload{
		maxSize:   maxAvatarSize,
		keyPrefix: "avatars",
		label:     "avatar",
		persist: func(userID, objectKey, url string) (*string, error) {
			_, oldKey, err := s.setAvatar(userID, objectKey, url)
			return oldKey, err
		},
		respond: func(c *gin.Context, url string) {
			c.JSON(http.StatusOK, avatarResponse{AvatarURL: url})
		},
	})
}

// deleteAvatarHandler supprime l'avatar courant (DELETE /users/me/avatar).
func (s *userService) deleteAvatarHandler(c *gin.Context) {
	s.handleImageDelete(c, s.clearAvatar, "avatar")
}

// uploadBannerHandler stocke une nouvelle bannière (PUT /users/me/banner).
func (s *userService) uploadBannerHandler(c *gin.Context) {
	s.handleImageUpload(c, imageUpload{
		maxSize:   maxBannerSize,
		keyPrefix: "banners",
		label:     "banner",
		persist:   s.setBanner,
		respond: func(c *gin.Context, url string) {
			c.JSON(http.StatusOK, bannerResponse{BannerURL: url})
		},
	})
}

// deleteBannerHandler supprime la bannière courante (DELETE /users/me/banner).
func (s *userService) deleteBannerHandler(c *gin.Context) {
	s.handleImageDelete(c, s.clearBanner, "banner")
}

// removeObject supprime un objet du bucket en best-effort (les échecs sont logués
// sans interrompre la requête : un fichier orphelin n'est pas bloquant).
func (s *userService) removeObject(objectKey string) {
	if err := s.mc.RemoveObject(context.Background(), s.cfg.Minio.Bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		log.Printf("failed to remove avatar object %s: %v", objectKey, err)
	}
}
