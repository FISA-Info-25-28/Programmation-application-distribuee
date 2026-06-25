package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authService struct {
	db          *gorm.DB
	cfg         authConfig
	rateLimiter rateLimiter
	mailer      mailer
	revoker     *sessionRevoker
}

var (
	errValidation         = errors.New("validation")
	errConflict           = errors.New("conflict")
	errInvalidCredentials = errors.New("invalid_credentials")
	errInvalidToken       = errors.New("invalid_token")
	errTokenExpired       = errors.New("token_expired")
	errEmailNotVerified   = errors.New("email_not_verified")
	errAccountSuspended   = errors.New("account_suspended")
	errSamePassword       = errors.New("same_password")
	errRefreshRevoked     = errors.New("refresh_revoked")
	errRefreshExpired     = errors.New("refresh_expired")
	errRefreshReuse       = errors.New("refresh_reuse")

	errMFAAlreadyEnabled = errors.New("mfa_already_enabled")
	errMFANotEnabled     = errors.New("mfa_not_enabled")
	errMFAInvalidCode    = errors.New("mfa_invalid_code")
)

type loginResult struct {
	Tokens       tokenPair
	MFARequired  bool
	PendingToken string
}

// defaultUserRole est le rôle attribué à un compte créé via l'inscription
// classique (cf. la valeur par défaut en base, authUserModel.Role).
const defaultUserRole = "user"

type tokenPair struct {
	AccessToken  string
	RefreshToken string
}

func newAuthService(db *gorm.DB, cfg authConfig) *authService {
	return &authService{
		db:          db,
		cfg:         cfg,
		rateLimiter: newRateLimiter(cfg),
		mailer:      newMailer(cfg),
		revoker:     newSessionRevoker(cfg),
	}
}

// register crée un compte NON vérifié et envoie le mail de confirmation. Aucune
// session n'est ouverte : le login restera refusé tant que l'email n'est pas
// vérifié (cf. errEmailNotVerified).
func (s *authService) register(username, email, password string, termsAccepted bool, termsVersion string) error {
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	if err := validateTermsAcceptance(termsAccepted, termsVersion); err != nil {
		return fmt.Errorf("%w: %v", errValidation, err)
	}
	if err := validateRegisterInput(username, email, password); err != nil {
		return fmt.Errorf("%w: %v", errValidation, err)
	}

	exists, err := s.usernameOrEmailExists(username, email)
	if err != nil {
		return err
	}
	if exists {
		return errConflict
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	termsAcceptedAt := time.Now().UTC()
	user := authUserModel{
		ID:              randomID("usr"),
		Username:        username,
		Email:           email,
		PasswordHash:    string(hashBytes),
		Role:            defaultUserRole,
		TermsAcceptedAt: &termsAcceptedAt,
		TermsVersion:    currentTermsVersion,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return err
	}

	// Synchro synchrone : si le user-service est indisponible, on compense en
	// supprimant le user auth pour ne pas laisser de compte auth sans profil.
	if err := s.syncUserProfile(user); err != nil {
		if delErr := s.db.Delete(&authUserModel{}, "id = ?", user.ID).Error; delErr != nil {
			log.Printf("auth register: compensation failed, orphan auth user %s (sync: %v, delete: %v)", user.ID, err, delErr)
		}
		return fmt.Errorf("sync user profile: %w", err)
	}

	// Envoi non bloquant : une panne SMTP transitoire ne doit pas faire échouer
	// l'inscription (le compte existe), l'utilisateur pourra demander un renvoi.
	if err := s.sendVerificationEmail(user); err != nil {
		log.Printf("auth register: failed to send verification email to user %s: %v", user.ID, err)
	}

	return nil
}

func (s *authService) login(identifier, password, ip, userAgent string) (loginResult, error) {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	password = strings.TrimSpace(password)
	if identifier == "" || password == "" {
		return loginResult{}, fmt.Errorf("%w: identifier and password are required", errValidation)
	}

	var user authUserModel
	if err := s.db.First(&user, "email = ? OR username = ?", identifier, identifier).Error; err != nil {
		return loginResult{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return loginResult{}, errInvalidCredentials
	}

	if err := validateSessionUser(user); err != nil {
		return loginResult{}, err
	}

	// If MFA enabled, issue a short-lived pending token instead of a full session.
	var mfa authMFAModel
	if err := s.db.First(&mfa, "user_id = ? AND enabled = true", user.ID).Error; err == nil {
		pendingToken, err := s.issuePendingMFAToken(user.ID, ip, userAgent)
		if err != nil {
			return loginResult{}, err
		}
		return loginResult{MFARequired: true, PendingToken: pendingToken}, nil
	}

	tokens, err := s.issueTokenPair(user, ip, userAgent)
	if err != nil {
		return loginResult{}, err
	}
	return loginResult{Tokens: tokens}, nil
}

const (
	accountStatusActive    = "active"
	accountStatusSuspended = "suspended"
	accountStatusBanned    = "banned"
)

func validateSessionUser(user authUserModel) error {
	if user.EmailVerifiedAt == nil {
		return errEmailNotVerified
	}
	if user.Status == accountStatusBanned || user.Status == accountStatusSuspended {
		return errAccountSuspended
	}
	return nil
}

func (s *authService) loadSessionUser(userID string) (authUserModel, error) {
	var user authUserModel
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return authUserModel{}, errInvalidToken
		}
		return authUserModel{}, err
	}
	if err := validateSessionUser(user); err != nil {
		return authUserModel{}, err
	}
	return user, nil
}

// setUserStatus met à jour l'état de modération d'un compte. Si le compte n'est
// plus actif, toutes ses sessions sont révoquées immédiatement : refresh tokens
// invalidés en base + borne de révocation publiée pour rejeter les access tokens
// déjà émis. Appelé par le user-service lors d'une sanction (endpoint interne).
func (s *authService) setUserStatus(userID, status string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errValidation
	}
	if status != accountStatusActive && status != accountStatusSuspended && status != accountStatusBanned {
		return errValidation
	}

	res := s.db.Model(&authUserModel{}).Where("id = ?", userID).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if status != accountStatusActive {
		now := time.Now().UTC()
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			return s.revokeAllUserRefreshTokens(tx, userID, now)
		}); err != nil {
			log.Printf("setUserStatus: échec révocation refresh tokens pour %s: %v", userID, err)
		}
		s.revoker.revokeUserSessions(userID)
	}
	return nil
}

// emailTokenModel regroupe les modèles de token email à usage unique. Ils sont
// manipulés de façon identique (invalidation des anciens + création du nouveau),
// d'où la contrainte générique partagée par invalidateAndCreateEmailToken.
type emailTokenModel interface {
	authEmailVerificationModel | authPasswordResetModel
}

// invalidateAndCreateEmailToken invalide, dans la transaction tx, les tokens non
// consommés de l'utilisateur puis crée le nouveau. Générique sur le type de
// modèle (vérification email / reset de mot de passe).
func invalidateAndCreateEmailToken[T emailTokenModel](tx *gorm.DB, userID string, now time.Time, record *T) error {
	if err := tx.Model(new(T)).
		Where("user_id = ? AND consumed_at IS NULL", userID).
		Update("consumed_at", now).Error; err != nil {
		return err
	}
	return tx.Create(record).Error
}

// issueEmailLink génère un token aléatoire à usage unique, le persiste via
// persist (qui reçoit le hash — le secret en clair n'est jamais stocké — et
// l'instant courant), construit le lien linkPath?token=… et l'envoie via send.
// Mutualise toute la logique des emails de vérification et de reset.
func (s *authService) issueEmailLink(
	email, linkPath string,
	persist func(tx *gorm.DB, tokenHash string, now time.Time) error,
	send func(email, link string) error,
) error {
	rawToken, err := randomToken()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return persist(tx, hashToken(rawToken), now)
	}); err != nil {
		return fmt.Errorf("persist email token: %w", err)
	}

	link := fmt.Sprintf("%s%s?token=%s", strings.TrimRight(s.cfg.AppBaseURL, "/"), linkPath, rawToken)
	if err := send(email, link); err != nil {
		return fmt.Errorf("send email %s: %w", linkPath, err)
	}
	return nil
}

// sendVerificationEmail invalide les éventuels tokens en cours pour cet
// utilisateur, en émet un nouveau (à usage unique, durée de vie courte) et
// envoie le lien par mail. Un seul lien est valide à la fois.
func (s *authService) sendVerificationEmail(user authUserModel) error {
	return s.issueEmailLink(user.Email, "/verify-email", func(tx *gorm.DB, tokenHash string, now time.Time) error {
		return invalidateAndCreateEmailToken(tx, user.ID, now, &authEmailVerificationModel{
			ID:        randomID("evt"),
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: now.Add(s.cfg.EmailVerificationTTL),
		})
	}, s.mailer.SendVerificationEmail)
}

// verifyEmail consomme le token (usage unique) et marque l'email comme vérifié.
func (s *authService) verifyEmail(rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return fmt.Errorf("%w: token is required", errValidation)
	}

	tokenHash := hashToken(rawToken)
	now := time.Now().UTC()

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var record authEmailVerificationModel
		if err := tx.First(&record, "token_hash = ?", tokenHash).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errInvalidToken
			}
			return err
		}

		if record.ConsumedAt != nil {
			return errInvalidToken
		}
		if now.After(record.ExpiresAt) {
			return errTokenExpired
		}

		// Consommation atomique : protège d'une double soumission concurrente.
		res := tx.Model(&authEmailVerificationModel{}).
			Where("id = ? AND consumed_at IS NULL", record.ID).
			Update("consumed_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errInvalidToken
		}

		// Idempotent : ne réécrit pas la date si déjà vérifié.
		return tx.Model(&authUserModel{}).
			Where("id = ? AND email_verified_at IS NULL", record.UserID).
			Update("email_verified_at", now).Error
	}); err != nil {
		return fmt.Errorf("verify email transaction: %w", err)
	}
	return nil
}

// resendVerification renvoie un lien de vérification. Pour éviter l'énumération
// d'emails, ne renvoie jamais d'erreur si l'email est inconnu ou déjà vérifié :
// le handler répond uniformément.
func (s *authService) resendVerification(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil
	}

	var user authUserModel
	if err := s.db.First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if user.EmailVerifiedAt != nil {
		return nil
	}

	return s.sendVerificationEmail(user)
}

// requestPasswordReset envoie un lien de réinitialisation. Comme resendVerification,
// ne révèle jamais l'existence du compte (anti-énumération) : le handler répond
// uniformément, on retourne nil si l'email est inconnu.
func (s *authService) requestPasswordReset(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil
	}

	var user authUserModel
	if err := s.db.First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	return s.sendPasswordResetEmail(user)
}

// sendPasswordResetEmail invalide les éventuels tokens de reset en cours, en émet
// un nouveau (usage unique, durée de vie courte) et envoie le lien par mail. Un
// seul lien est valide à la fois. Calqué sur sendVerificationEmail.
func (s *authService) sendPasswordResetEmail(user authUserModel) error {
	return s.issueEmailLink(user.Email, "/reset-password", func(tx *gorm.DB, tokenHash string, now time.Time) error {
		return invalidateAndCreateEmailToken(tx, user.ID, now, &authPasswordResetModel{
			ID:        randomID("prt"),
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: now.Add(s.cfg.PasswordResetTTL),
		})
	}, s.mailer.SendPasswordResetEmail)
}

// loadResetTarget récupère le token de réinitialisation et vérifie qu'il est
// exploitable (présent, non consommé, non expiré), puis charge l'utilisateur
// associé. Extrait de resetPassword pour en contenir la complexité.
func loadResetTarget(tx *gorm.DB, tokenHash string, now time.Time) (authPasswordResetModel, authUserModel, error) {
	var record authPasswordResetModel
	if err := tx.First(&record, "token_hash = ?", tokenHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return record, authUserModel{}, errInvalidToken
		}
		return record, authUserModel{}, err
	}

	if record.ConsumedAt != nil {
		return record, authUserModel{}, errInvalidToken
	}
	if now.After(record.ExpiresAt) {
		return record, authUserModel{}, errTokenExpired
	}

	var user authUserModel
	if err := tx.First(&user, "id = ?", record.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return record, user, errInvalidToken
		}
		return record, user, err
	}
	return record, user, nil
}

// resetPassword consomme le token (usage unique), remplace le mot de passe et
// révoque toutes les sessions. Tout est atomique : un token consommé ne peut pas
// laisser un mot de passe à moitié changé.
func (s *authService) resetPassword(rawToken, newPassword string) error {
	rawToken = strings.TrimSpace(rawToken)
	newPassword = strings.TrimSpace(newPassword)
	if rawToken == "" {
		return fmt.Errorf("%w: token is required", errValidation)
	}

	tokenHash := hashToken(rawToken)
	now := time.Now().UTC()

	var userEmail, revokedUserID string
	// validationErr est capturé à part : renvoyé tel quel (sans le wrapping
	// "reset password transaction:"), il garde un message propre côté API.
	var validationErr error
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		record, user, err := loadResetTarget(tx, tokenHash, now)
		if err != nil {
			return err
		}
		userEmail = user.Email
		revokedUserID = user.ID

		// Validation avant consommation du token : un mot de passe invalide ne doit
		// pas brûler le lien de réinitialisation. On dispose ici de l'identité du
		// compte pour interdire un mot de passe contenant le username ou l'email.
		if err := validatePassword(newPassword, user.Username, user.Email); err != nil {
			validationErr = fmt.Errorf("%w: %v", errValidation, err)
			return validationErr
		}

		// Refuse de réutiliser le mot de passe courant.
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)) == nil {
			return errSamePassword
		}

		// Consommation atomique : protège d'une double soumission concurrente.
		res := tx.Model(&authPasswordResetModel{}).
			Where("id = ? AND consumed_at IS NULL", record.ID).
			Update("consumed_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errInvalidToken
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		return s.applyNewPassword(tx, user.ID, string(hashed), now)
	}); err != nil {
		if validationErr != nil {
			return validationErr
		}
		return fmt.Errorf("reset password transaction: %w", err)
	}

	// Borne de révocation : invalide immédiatement les access tokens des autres
	// sessions au niveau du gateway (les refresh tokens sont déjà révoqués en base).
	s.revoker.revokeUserSessions(revokedUserID)

	// Notification non bloquante : l'opération a réussi même si l'envoi échoue.
	if err := s.mailer.SendPasswordChangedEmail(userEmail); err != nil {
		log.Printf("auth reset-password: failed to send notification to %s: %v", userEmail, err)
	}
	return nil
}

// changePassword change le mot de passe d'un utilisateur connecté après
// vérification du mot de passe actuel, puis révoque toutes les sessions.
func (s *authService) changePassword(userID, currentPassword, newPassword string) error {
	userID = strings.TrimSpace(userID)
	currentPassword = strings.TrimSpace(currentPassword)
	newPassword = strings.TrimSpace(newPassword)
	if userID == "" {
		return fmt.Errorf("%w: user is required", errValidation)
	}
	if currentPassword == "" {
		return fmt.Errorf("%w: current password is required", errValidation)
	}

	var user authUserModel
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	// Validation après le fetch : permet d'interdire un mot de passe contenant le
	// username ou l'email du compte.
	if err := validatePassword(newPassword, user.Username, user.Email); err != nil {
		return fmt.Errorf("%w: %v", errValidation, err)
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return errInvalidCredentials
	}
	// Refuse un nouveau mot de passe identique à l'actuel.
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)) == nil {
		return errSamePassword
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.applyNewPassword(tx, user.ID, string(hashed), now)
	}); err != nil {
		return fmt.Errorf("change password transaction: %w", err)
	}

	// Borne de révocation : invalide immédiatement les access tokens des autres
	// sessions au niveau du gateway (les refresh tokens sont déjà révoqués en base).
	s.revoker.revokeUserSessions(user.ID)

	// Notification non bloquante : l'opération a réussi même si l'envoi échoue.
	if err := s.mailer.SendPasswordChangedEmail(user.Email); err != nil {
		log.Printf("auth change-password: failed to send notification to %s: %v", user.Email, err)
	}
	return nil
}

func (s *authService) me(userID string) (*authUserModel, error) {
	var user authUserModel
	if err := s.db.First(&user, "id = ?", strings.TrimSpace(userID)).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *authService) refresh(refreshToken, ip, userAgent string) (tokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return tokenPair{}, fmt.Errorf("%w: refreshToken is required", errValidation)
	}

	refreshHash := hashToken(refreshToken)
	now := time.Now().UTC()

	var stored authRefreshTokenModel
	if err := s.db.First(&stored, "token_hash = ?", refreshHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tokenPair{}, errInvalidToken
		}
		return tokenPair{}, err
	}

	if stored.RevokedAt != nil {
		if err := s.revokeRefreshFamily(stored.FamilyID, now); err != nil {
			return tokenPair{}, err
		}
		return tokenPair{}, errRefreshReuse
	}
	if now.After(stored.ExpiresAt) {
		return tokenPair{}, errRefreshExpired
	}

	user, err := s.loadSessionUser(stored.UserID)
	if err != nil {
		return tokenPair{}, err
	}

	pair := tokenPair{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&authRefreshTokenModel{}).
			Where("id = ? AND revoked_at IS NULL", stored.ID).
			Update("revoked_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errRefreshRevoked
		}

		newRefreshToken, err := randomToken()
		if err != nil {
			return err
		}

		newRecord := authRefreshTokenModel{
			ID:            randomID("rt"),
			UserID:        user.ID,
			SessionID:     stored.SessionID,
			FamilyID:      stored.FamilyID,
			RotatedFromID: stored.ID,
			TokenHash:     hashToken(newRefreshToken),
			ExpiresAt:     now.Add(s.cfg.RefreshTTL),
			LastSeenAt:    &now,
			LastIP:        strings.TrimSpace(ip),
			UserAgent:     trimUserAgent(userAgent),
		}
		if err := tx.Create(&newRecord).Error; err != nil {
			return err
		}

		accessToken, err := signJWT(user.ID, user.Role, user.Username, s.cfg, s.cfg.TokenTTL)
		if err != nil {
			return err
		}

		pair = tokenPair{AccessToken: accessToken, RefreshToken: newRefreshToken}
		return nil
	})
	if err != nil {
		return tokenPair{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	return pair, nil
}

func (s *authService) logout(refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return fmt.Errorf("%w: refreshToken is required", errValidation)
	}

	now := time.Now().UTC()
	refreshHash := hashToken(refreshToken)
	var stored authRefreshTokenModel
	if err := s.db.First(&stored, "token_hash = ?", refreshHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if err := s.db.Model(&authRefreshTokenModel{}).
		Where("session_id = ? AND revoked_at IS NULL", stored.SessionID).
		Update("revoked_at", now).Error; err != nil {
		return err
	}

	return nil
}

func (s *authService) issueTokenPair(user authUserModel, ip, userAgent string) (tokenPair, error) {
	accessToken, err := signJWT(user.ID, user.Role, user.Username, s.cfg, s.cfg.TokenTTL)
	if err != nil {
		return tokenPair{}, err
	}

	refreshToken, err := randomToken()
	if err != nil {
		return tokenPair{}, err
	}

	sessionID := randomID("sess")
	familyID := randomID("fam")
	now := time.Now().UTC()

	refreshRecord := authRefreshTokenModel{
		ID:         randomID("rt"),
		UserID:     user.ID,
		SessionID:  sessionID,
		FamilyID:   familyID,
		TokenHash:  hashToken(refreshToken),
		ExpiresAt:  now.Add(s.cfg.RefreshTTL),
		LastSeenAt: &now,
		LastIP:     strings.TrimSpace(ip),
		UserAgent:  trimUserAgent(userAgent),
	}
	if err := s.db.Create(&refreshRecord).Error; err != nil {
		return tokenPair{}, err
	}

	return tokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *authService) revokeRefreshFamily(familyID string, now time.Time) error {
	if strings.TrimSpace(familyID) == "" {
		return nil
	}
	if err := s.db.Model(&authRefreshTokenModel{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error; err != nil {
		return err
	}
	return nil
}

// applyNewPassword écrit le nouveau hash et révoque toutes les sessions de
// l'utilisateur de façon atomique. À appeler dans une transaction (tx).
func (s *authService) applyNewPassword(tx *gorm.DB, userID, hashedPassword string, now time.Time) error {
	if err := tx.Model(&authUserModel{}).
		Where("id = ?", userID).
		Update("password_hash", hashedPassword).Error; err != nil {
		return err
	}
	return s.revokeAllUserRefreshTokens(tx, userID, now)
}

// revokeAllUserRefreshTokens révoque toutes les sessions actives de l'utilisateur
// (toutes familles confondues). Utilisé après un changement de mot de passe.
func (s *authService) revokeAllUserRefreshTokens(tx *gorm.DB, userID string, now time.Time) error {
	return tx.Model(&authRefreshTokenModel{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

type mfaSetupResult struct {
	Secret      string
	URI         string
	BackupCodes []string
}

func (s *authService) setupMFA(userID string) (mfaSetupResult, error) {
	user, err := s.loadSessionUser(userID)
	if err != nil {
		return mfaSetupResult{}, err
	}

	var existing authMFAModel
	if err := s.db.First(&existing, "user_id = ?", userID).Error; err == nil && existing.Enabled {
		return mfaSetupResult{}, errMFAAlreadyEnabled
	}

	secret, err := generateTOTPSecret()
	if err != nil {
		return mfaSetupResult{}, err
	}
	backupCodes, err := generateBackupCodes(mfaBackupCodeCount)
	if err != nil {
		return mfaSetupResult{}, err
	}

	now := time.Now().UTC()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		tx.Delete(&authMFAModel{}, "user_id = ?", userID)
		tx.Delete(&authMFABackupCodeModel{}, "user_id = ?", userID)
		if err := tx.Create(&authMFAModel{
			ID:         randomID("mfa"),
			UserID:     userID,
			TOTPSecret: secret,
			Enabled:    false,
			CreatedAt:  now,
			UpdatedAt:  now,
		}).Error; err != nil {
			return err
		}
		for _, code := range backupCodes {
			if err := tx.Create(&authMFABackupCodeModel{
				ID:        randomID("bc"),
				UserID:    userID,
				CodeHash:  hashToken(code),
				CreatedAt: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return mfaSetupResult{}, fmt.Errorf("setup MFA transaction: %w", err)
	}

	uri := totpURI("Breezy", user.Email, secret)
	return mfaSetupResult{Secret: secret, URI: uri, BackupCodes: backupCodes}, nil
}

func (s *authService) confirmMFA(userID, totpCode string) error {
	var mfa authMFAModel
	if err := s.db.First(&mfa, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errMFANotEnabled
		}
		return err
	}
	if mfa.Enabled {
		return errMFAAlreadyEnabled
	}
	if !verifyTOTP(mfa.TOTPSecret, totpCode) {
		return errMFAInvalidCode
	}
	return s.db.Model(&authMFAModel{}).Where("user_id = ?", userID).Update("enabled", true).Error
}

func (s *authService) disableMFA(userID, code string) error {
	var mfa authMFAModel
	if err := s.db.First(&mfa, "user_id = ? AND enabled = true", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errMFANotEnabled
		}
		return err
	}
	if !verifyTOTP(mfa.TOTPSecret, code) && !s.consumeBackupCode(userID, code) {
		return errMFAInvalidCode
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&authMFAModel{}, "user_id = ?", userID).Error; err != nil {
			return err
		}
		return tx.Delete(&authMFABackupCodeModel{}, "user_id = ?", userID).Error
	}); err != nil {
		return fmt.Errorf("disable MFA transaction: %w", err)
	}
	return nil
}

func (s *authService) consumeBackupCode(userID, code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	now := time.Now().UTC()
	res := s.db.Model(&authMFABackupCodeModel{}).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", userID, hashToken(code)).
		Update("used_at", now)
	return res.Error == nil && res.RowsAffected > 0
}

func (s *authService) issuePendingMFAToken(userID, ip, userAgent string) (string, error) {
	rawToken, err := randomToken()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := s.db.Create(&authMFAPendingModel{
		ID:        randomID("mfap"),
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: now.Add(mfaPendingTTL),
		IP:        strings.TrimSpace(ip),
		UserAgent: trimUserAgent(userAgent),
		CreatedAt: now,
	}).Error; err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *authService) verifyMFALogin(pendingToken, code, ip, userAgent string) (tokenPair, error) {
	pendingToken = strings.TrimSpace(pendingToken)
	code = strings.TrimSpace(code)
	if pendingToken == "" || code == "" {
		return tokenPair{}, fmt.Errorf("%w: pendingToken and code are required", errValidation)
	}

	tokenHash := hashToken(pendingToken)
	now := time.Now().UTC()

	var pending authMFAPendingModel
	if err := s.db.First(&pending, "token_hash = ?", tokenHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tokenPair{}, errInvalidToken
		}
		return tokenPair{}, err
	}
	if now.After(pending.ExpiresAt) {
		return tokenPair{}, errTokenExpired
	}

	// Consume pending token (single use)
	if err := s.db.Delete(&authMFAPendingModel{}, "id = ?", pending.ID).Error; err != nil {
		return tokenPair{}, err
	}

	user, err := s.loadSessionUser(pending.UserID)
	if err != nil {
		return tokenPair{}, err
	}

	var mfa authMFAModel
	if err := s.db.First(&mfa, "user_id = ? AND enabled = true", user.ID).Error; err != nil {
		return tokenPair{}, errMFANotEnabled
	}

	if !verifyTOTP(mfa.TOTPSecret, code) && !s.consumeBackupCode(user.ID, strings.ToUpper(code)) {
		return tokenPair{}, errMFAInvalidCode
	}

	return s.issueTokenPair(user, ip, userAgent)
}

func (s *authService) usernameOrEmailExists(username, email string) (bool, error) {
	var count int64
	if err := s.db.Model(&authUserModel{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func randomID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return prefix + "_fallback"
	}
	return prefix + "_" + hex.EncodeToString(buf)
}

// randomToken génère un token aléatoire de 32 octets encodé en hexadécimal.
// Sert aux refresh tokens et aux tokens de réinitialisation / vérification email.
func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random token bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// hashToken renvoie le SHA-256 hex d'un token. Sert pour les refresh tokens et
// les tokens de vérification email : on ne stocke jamais le secret en clair.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func trimUserAgent(userAgent string) string {
	trimmed := strings.TrimSpace(userAgent)
	if len(trimmed) <= 512 {
		return trimmed
	}
	return trimmed[:512]
}
