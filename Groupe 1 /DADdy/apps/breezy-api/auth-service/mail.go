package main

import (
	"html"
	"log"
	"mime"
	"mime/quotedprintable"
	"strconv"
	"strings"
	"time"
)

// mailer envoie les emails transactionnels du service. Abstrait pour permettre
// une impl SMTP en prod et une impl "log" sans dépendance en dev/test.
type mailer interface {
	SendVerificationEmail(to, link string) error
	SendPasswordResetEmail(to, link string) error
	SendPasswordChangedEmail(to string) error
}

// newMailer choisit l'implémentation selon la config : SMTP si SMTP_HOST est
// défini, sinon un mailer qui journalise le lien (dev, `docker compose up` sans
// serveur mail).
func newMailer(cfg authConfig) mailer {
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		log.Print("auth-service: SMTP_HOST non défini, mailer en mode log (les liens de vérification sont écrits dans les logs)")
		return logMailer{}
	}
	return smtpMailer{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     cfg.MailFrom,
	}
}

// ── Impl dev : journalise le lien ───────────────────────────────────────────

type logMailer struct{}

func (logMailer) SendVerificationEmail(to, link string) error {
	log.Printf("auth-service [DEV MAIL] verification email to %s: %s", to, link)
	return nil
}

func (logMailer) SendPasswordResetEmail(to, link string) error {
	log.Printf("auth-service [DEV MAIL] password reset email to %s: %s", to, link)
	return nil
}

func (logMailer) SendPasswordChangedEmail(to string) error {
	log.Printf("auth-service [DEV MAIL] password changed notification to %s", to)
	return nil
}

// buildMIMEMessage assemble un message RFC 5322 en multipart/alternative (CRLF) :
// une partie texte (fallback / anti-spam) et une partie HTML. Les clients
// affichent la dernière partie qu'ils savent rendre, donc le HTML vient en second.
func buildMIMEMessage(from, to, subject, htmlBody, textBody string) []byte {
	boundary := "breezy-" + strconv.FormatInt(time.Now().UnixNano(), 36)

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + encodeHeaderWord(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	b.WriteString(encodeQuotedPrintable(textBody))
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	b.WriteString(encodeQuotedPrintable(htmlBody))
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

// encodeQuotedPrintable encode un corps en quoted-printable (RFC 2045). Plus sûr
// que 8bit pour les caractères non-ASCII : 7-bit, traversé sans altération par
// les relais qui n'annoncent pas l'extension 8BITMIME.
func encodeQuotedPrintable(s string) string {
	var b strings.Builder
	w := quotedprintable.NewWriter(&b)
	// Écriture sur un strings.Builder : Write/Close ne renvoient pas d'erreur réelle.
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return b.String()
}

// encodeHeaderWord encode un en-tête en MIME "encoded-word" (RFC 2047) dès qu'il
// contient des caractères non-ASCII, pour que les accents s'affichent partout.
func encodeHeaderWord(s string) string {
	for _, r := range s {
		if r > 127 {
			return mime.QEncoding.Encode("UTF-8", s)
		}
	}
	return s
}

// verificationEmailText est le fallback texte brut de l'email de vérification.
func verificationEmailText(link string) string {
	return "Bienvenue sur Breezy !\r\n\r\n" +
		"Confirmez votre adresse email en ouvrant ce lien :\r\n" +
		link + "\r\n\r\n" +
		"Ce lien expire prochainement. Si vous n'êtes pas à l'origine de cette " +
		"inscription, vous pouvez ignorer ce message."
}

// verificationEmailHTML rend l'email de vérification. Layout en tables et styles
// inline (contrainte des clients mail), aucune ressource externe, bouton
// "bulletproof" entouré d'un lien texte de secours.
func verificationEmailHTML(link string) string {
	safeLink := html.EscapeString(link)
	return emailShell(
		`<h1 style="margin:0 0 16px; font-size:20px; font-weight:700; color:#0f172a;">Bienvenue sur Breezy&nbsp;!</h1>
              <p style="margin:0 0 24px; font-size:15px; line-height:1.6; color:#475569;">
                Plus qu'une étape&nbsp;: confirmez votre adresse email pour activer votre compte et commencer à publier.
              </p>` +
			emailButton("Vérifier mon adresse email", safeLink) +
			emailFallbackLink(safeLink) +
			`<p style="margin:0; font-size:13px; line-height:1.6; color:#94a3b8;">
                Ce lien expire prochainement. Si vous n'êtes pas à l'origine de cette inscription, ignorez simplement ce message.
              </p>`,
	)
}

// ── Templates : réinitialisation de mot de passe ─────────────────────────────

// passwordResetEmailText est le fallback texte brut de l'email de réinitialisation.
func passwordResetEmailText(link string) string {
	return "Réinitialisation de votre mot de passe\r\n\r\n" +
		"Vous avez demandé à réinitialiser le mot de passe de votre compte Breezy. " +
		"Ouvrez ce lien pour en choisir un nouveau :\r\n" +
		link + "\r\n\r\n" +
		"Ce lien expire prochainement et ne peut servir qu'une fois. Si vous n'êtes " +
		"pas à l'origine de cette demande, ignorez ce message : votre mot de passe " +
		"reste inchangé."
}

func passwordResetEmailHTML(link string) string {
	safeLink := html.EscapeString(link)
	return emailShell(
		`<h1 style="margin:0 0 16px; font-size:20px; font-weight:700; color:#0f172a;">Réinitialisation du mot de passe</h1>
              <p style="margin:0 0 24px; font-size:15px; line-height:1.6; color:#475569;">
                Vous avez demandé à réinitialiser le mot de passe de votre compte. Cliquez sur le bouton ci-dessous pour en choisir un nouveau.
              </p>` +
			emailButton("Réinitialiser mon mot de passe", safeLink) +
			emailFallbackLink(safeLink) +
			`<p style="margin:0; font-size:13px; line-height:1.6; color:#94a3b8;">
                Ce lien expire prochainement et ne peut servir qu'une fois. Si vous n'êtes pas à l'origine de cette demande, ignorez ce message&nbsp;: votre mot de passe reste inchangé.
              </p>`,
	)
}

// ── Templates : notification de changement de mot de passe ───────────────────

// passwordChangedEmailText est le fallback texte brut de la notification de
// changement de mot de passe (aucun lien : c'est une alerte de sécurité).
func passwordChangedEmailText() string {
	return "Votre mot de passe a été modifié\r\n\r\n" +
		"Le mot de passe de votre compte Breezy vient d'être modifié et toutes vos " +
		"sessions ont été déconnectées.\r\n\r\n" +
		"Si vous n'êtes pas à l'origine de ce changement, réinitialisez immédiatement " +
		"votre mot de passe via « mot de passe oublié » et sécurisez votre boîte mail."
}

func passwordChangedEmailHTML() string {
	return emailShell(
		`<h1 style="margin:0 0 16px; font-size:20px; font-weight:700; color:#0f172a;">Votre mot de passe a été modifié</h1>
              <p style="margin:0 0 24px; font-size:15px; line-height:1.6; color:#475569;">
                Le mot de passe de votre compte Breezy vient d'être modifié et toutes vos sessions ont été déconnectées.
              </p>
              <p style="margin:0; font-size:13px; line-height:1.6; color:#94a3b8;">
                Si vous n'êtes pas à l'origine de ce changement, réinitialisez immédiatement votre mot de passe via «&nbsp;mot de passe oublié&nbsp;» et sécurisez votre boîte mail.
              </p>`,
	)
}

// ── Coquille HTML commune ────────────────────────────────────────────────────

// emailButton rend un bouton "bulletproof" (table + lien) pour les clients mail.
func emailButton(label, safeLink string) string {
	return `<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 0 24px;">
                <tr>
                  <td align="center" style="border-radius:12px; background-color:#6366f1;">
                    <a href="` + safeLink + `" style="display:inline-block; padding:14px 28px; font-size:15px; font-weight:600; color:#ffffff; text-decoration:none; border-radius:12px;">
                      ` + label + `
                    </a>
                  </td>
                </tr>
              </table>`
}

// emailFallbackLink rend le lien texte de secours affiché sous le bouton.
func emailFallbackLink(safeLink string) string {
	return `<p style="margin:0 0 8px; font-size:13px; line-height:1.6; color:#64748b;">
                Si le bouton ne fonctionne pas, copiez-collez ce lien dans votre navigateur&nbsp;:
              </p>
              <p style="margin:0 0 24px; font-size:13px; line-height:1.6; word-break:break-all;">
                <a href="` + safeLink + `" style="color:#6366f1; text-decoration:underline;">` + safeLink + `</a>
              </p>`
}

// emailShell assemble la coquille commune des mails transactionnels (header
// Breezy, carte centrée, footer). innerHTML est le contenu déjà sûr de la carte.
// Layout en tables et styles inline (contrainte des clients mail), sans ressource
// externe.
func emailShell(innerHTML string) string {
	return `<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="color-scheme" content="light dark">
</head>
<body style="margin:0; padding:0; background-color:#f1f5f9; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f1f5f9; padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:480px; background-color:#ffffff; border-radius:16px; overflow:hidden; box-shadow:0 1px 3px rgba(15,23,42,0.08);">
          <tr>
            <td style="background:linear-gradient(135deg,#6366f1,#8b5cf6); background-color:#6366f1; padding:32px 40px;">
              <span style="color:#ffffff; font-size:24px; font-weight:700; letter-spacing:-0.02em;">Breezy</span>
            </td>
          </tr>
          <tr>
            <td style="padding:40px;">
              ` + innerHTML + `
            </td>
          </tr>
          <tr>
            <td style="padding:20px 40px; background-color:#f8fafc; border-top:1px solid #e2e8f0;">
              <p style="margin:0; font-size:12px; color:#94a3b8;">Breezy — email automatique, merci de ne pas y répondre.</p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
}
