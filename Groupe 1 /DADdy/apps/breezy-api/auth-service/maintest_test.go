//go:build integration

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/shared"
)

// sharedDSN pointe vers le Postgres jetable démarré une fois pour le paquet.
var sharedDSN string

// TestMain démarre un Postgres via testcontainers, applique les migrations GORM
// (les mêmes que main.go) puis lance la suite.
func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("daddy_auth_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("connection string: " + err.Error())
	}
	sharedDSN = dsn

	db, err := shared.ConnectPostgres(dsn)
	if err != nil {
		panic("connect postgres: " + err.Error())
	}
	if err := shared.AutoMigrate(db, &authUserModel{}, &authRefreshTokenModel{}, &authEmailVerificationModel{}, &authPasswordResetModel{},
		&authMFAModel{}, &authMFABackupCodeModel{}, &authMFAPendingModel{}, &authSocialAccountModel{}, &authOAuthExchangeModel{}); err != nil {
		panic("migrate: " + err.Error())
	}

	code := m.Run()

	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

// testConfig construit une config 100% locale : pas d'URL de service (synchro
// register court-circuitée), SMTP vide (mailer log), Valkey vide (rate limiter
// mémoire + revoker no-op). La clé de signature JWT est préparée.
func testConfig() authConfig {
	cfg := authConfig{
		JWTSecret:            "test-secret-key",
		JWTAlg:               "HS256",
		JWTIssuer:            "daddy-auth-test",
		JWTAudience:          "daddy-api-test",
		JWTKID:               "v1",
		TokenTTL:             15 * time.Minute,
		RefreshTTL:           30 * 24 * time.Hour,
		EmailVerificationTTL: 24 * time.Hour,
		PasswordResetTTL:     time.Hour,
		AppBaseURL:           "http://localhost:5173",
		MailFrom:             "Breezy <no-reply@breezy.local>",
		InternalKey:          "test-internal-key",
	}
	prepareAuthSigningKey(&cfg)
	return cfg
}

// newTestService ouvre une connexion neuve, vide les tables, et renvoie un
// authService prêt à l'emploi (rate limiter mémoire, mailer log, revoker no-op).
func newTestService(t *testing.T) *authService {
	t.Helper()
	db, err := shared.ConnectPostgres(sharedDSN)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	truncateAll(t, db)
	return newAuthService(db, testConfig())
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE auth_user_models, auth_refresh_token_models, auth_email_verification_models, auth_password_reset_models, auth_mfa_models, auth_mfa_backup_code_models, auth_mfa_pending_models, auth_social_account_models, auth_o_auth_exchange_models RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedVerifiedUser insère un compte vérifié et actif avec un mot de passe haché,
// et renvoie son ID. Utile pour les tests login/refresh/changePassword qui
// présupposent un compte connectable.
func seedVerifiedUser(t *testing.T, s *authService, username, email, password, status string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	now := time.Now().UTC()
	u := authUserModel{
		ID:              randomID("usr"),
		Username:        username,
		Email:           email,
		PasswordHash:    string(hash),
		Role:            defaultUserRole,
		Status:          status,
		EmailVerifiedAt: &now,
		TermsAcceptedAt: &now,
		TermsVersion:    currentTermsVersion,
	}
	if err := s.db.Create(&u).Error; err != nil {
		t.Fatalf("seed user %s: %v", username, err)
	}
	return u.ID
}

// loadUser relit un compte par ID.
func loadUser(t *testing.T, s *authService, id string) authUserModel {
	t.Helper()
	var u authUserModel
	if err := s.db.First(&u, "id = ?", id).Error; err != nil {
		t.Fatalf("load user %s: %v", id, err)
	}
	return u
}
