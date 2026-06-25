package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/shared"
)

// Ce fichier regroupe le code d'infrastructure non couvrable par les tests :
// point d'entrée (main), configuration MinIO, migrations DB et upload d'objets
// vers MinIO (réseau/stockage externe). Isolé ici pour être exclu du calcul de
// couverture (voir .coverignore). La logique HTTP/métier testable reste dans
// main.go.

func loadMinioCfg() minioCfg {
	return minioCfg{
		Endpoint:  shared.GetEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: shared.GetEnv("MINIO_ACCESS_KEY", defaultMinioUser),
		SecretKey: shared.GetEnv("MINIO_SECRET_KEY", defaultMinioUser),
		Bucket:    shared.GetEnv("MINIO_BUCKET", "breezy-media"),
		PublicURL: shared.GetEnv("MINIO_PUBLIC_URL", "http://localhost:9000"),
	}
}

func ensureMediaTypeConstraint(db *gorm.DB) error {
	if err := db.Exec("ALTER TABLE media DROP CONSTRAINT IF EXISTS chk_media_type").Error; err != nil {
		return fmt.Errorf("drop media type constraint: %w", err)
	}
	if err := db.Exec("ALTER TABLE media ADD CONSTRAINT chk_media_type CHECK (media_type IN ('image','video','audio'))").Error; err != nil {
		return fmt.Errorf("add media type constraint: %w", err)
	}
	return nil
}

func abortMultipartError(c *gin.Context, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		shared.AbortError(c, http.StatusRequestEntityTooLarge, shared.ErrValidation, maxMediaSizeText)
		return
	}
	shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid multipart body")
}

func closeMultipartPart(part *multipart.Part, label string) {
	if err := part.Close(); err != nil {
		log.Printf("failed to close %s: %v", label, err)
	}
}

func nextMediaPart(reader *multipart.Reader) (*multipart.Part, bool, error) {
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("read multipart part: %w", err)
		}
		if part.FormName() == "file" {
			return part, true, nil
		}
		closeMultipartPart(part, "ignored multipart field")
	}
}

func (a *app) storeMediaUpload(ctx context.Context, upload mediaUploadPart, uploaderID string) (mediaModel, error) {
	fileReader := &maxFileReader{reader: upload.part, max: maxMediaSize}
	_, err := a.mc.PutObject(ctx, a.mcfg.Bucket, upload.objectKey, fileReader, -1, minio.PutObjectOptions{
		ContentType: upload.contentType,
		PartSize:    mediaPartSize,
	})
	if err != nil {
		return mediaModel{}, fmt.Errorf("put media object: %w", err)
	}

	objectURL := fmt.Sprintf("%s/%s/%s", a.mcfg.PublicURL, a.mcfg.Bucket, upload.objectKey)
	media := mediaModel{ObjectKey: upload.objectKey, MediaType: upload.mediaType, URL: objectURL, UploaderID: uploaderID}
	if err := a.db.Create(&media).Error; err != nil {
		return mediaModel{}, fmt.Errorf("save media record: %w", err)
	}
	return media, nil
}

func abortMediaStoreError(c *gin.Context, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.Is(err, errMediaTooLarge) || errors.As(err, &maxBytesErr) {
		shared.AbortError(c, http.StatusRequestEntityTooLarge, shared.ErrValidation, maxMediaSizeText)
		return
	}
	log.Printf("media upload error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to store file"})
}

func (a *app) uploadMedia(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxMultipartSize)

	reader, err := c.Request.MultipartReader()
	if err != nil {
		abortMultipartError(c, err)
		return
	}

	part, found, err := nextMediaPart(reader)
	if err != nil {
		abortMultipartError(c, err)
		return
	}
	if !found {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgFileRequired)
		return
	}
	defer closeMultipartPart(part, "uploaded file part")

	upload, err := buildMediaUploadPart(part)
	if errors.Is(err, errMediaFileRequired) {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgFileRequired)
		return
	}
	if errors.Is(err, errUnsupportedMedia) {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "file must be an image, video, or audio")
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to prepare file"})
		return
	}

	media, err := a.storeMediaUpload(c.Request.Context(), upload, a.callerID(c))
	if err != nil {
		abortMediaStoreError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{keyData: mediaResponse{ID: media.ID, MediaType: media.MediaType, URL: media.URL}})
}

func ensureBucket(mc *minio.Client, bucket string) error {
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket existence: %w", err)
	}
	if exists {
		return nil
	}
	if err := mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucket)
	if err := mc.SetBucketPolicy(ctx, bucket, policy); err != nil {
		return fmt.Errorf("set bucket policy: %w", err)
	}
	return nil
}

func main() {
	port := shared.GetEnv("PORT", "3103")
	postgresDSN := shared.GetEnvAny([]string{"POST_DATABASE_URL", "DATABASE_URL"}, "postgres://postgres:postgres@localhost:5432/daddy?sslmode=disable")

	db, err := shared.ConnectPostgres(postgresDSN)
	if err != nil {
		log.Fatalf("failed to connect post-service database: %v", err)
	}
	if err := shared.AutoMigrate(db, &postModel{}, &likeModel{}, &rebreezersModel{}, &mediaModel{}, &hashtagModel{}, &postHashtagModel{}, &postMentionModel{}, &pollModel{}, &pollOptionModel{}, &pollVoteModel{}, &reportModel{}, &bookmarkModel{}, &userAffinityModel{}, &hashtagNeighborModel{}, &hashtagClusterModel{}); err != nil {
		log.Fatalf("failed to migrate post-service database: %v", err)
	}
	if err := ensureMediaTypeConstraint(db); err != nil {
		log.Fatalf("failed to migrate media type constraint: %v", err)
	}

	mcfg := loadMinioCfg()
	mc, err := minio.New(mcfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(mcfg.AccessKey, mcfg.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("failed to connect MinIO: %v", err)
	}
	if err := ensureBucket(mc, mcfg.Bucket); err != nil {
		log.Fatalf("failed to ensure MinIO bucket: %v", err)
	}

	a := &app{
		db:          db,
		mc:          mc,
		mcfg:        mcfg,
		notifURL:    shared.GetEnv("NOTIF_SERVICE_URL", ""),
		userURL:     shared.GetEnv("USER_SERVICE_URL", ""),
		internalKey: shared.SecretEnv("INTERNAL_API_KEY", "dev-internal-key"),
	}

	// Oubli progressif des préférences : érode les poids d'affinité une fois par
	// jour pour que le feed Pour Toi suive l'évolution des centres d'intérêt.
	a.startAffinityDecay(24*time.Hour, 0.97)

	// Déduction comportementale du thème des breezes sans hashtag, d'après les
	// centres d'intérêt de leurs interactants (raffinée à chaque passe).
	a.startThemeInference(time.Minute)

	router := newRouter(a)

	log.Printf("post-service listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
