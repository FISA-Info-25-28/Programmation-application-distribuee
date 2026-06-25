package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"time"
)

// smtpMailer envoie les emails transactionnels via un serveur SMTP. Code I/O
// réseau, non couvert par les tests : isolé ici pour être exclu du calcul de
// couverture (voir .coverignore). Les gabarits MIME purs vivent dans mail.go.
type smtpMailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

const smtpDialTimeout = 10 * time.Second

func (m smtpMailer) SendVerificationEmail(to, link string) error {
	subject := "Vérifiez votre adresse email"
	return m.send(to, subject, verificationEmailHTML(link), verificationEmailText(link))
}

func (m smtpMailer) SendPasswordResetEmail(to, link string) error {
	subject := "Réinitialisation de votre mot de passe"
	return m.send(to, subject, passwordResetEmailHTML(link), passwordResetEmailText(link))
}

func (m smtpMailer) SendPasswordChangedEmail(to string) error {
	subject := "Votre mot de passe a été modifié"
	return m.send(to, subject, passwordChangedEmailHTML(), passwordChangedEmailText())
}

func (m smtpMailer) send(to, subject, htmlBody, textBody string) error {
	msg := buildMIMEMessage(m.from, to, subject, htmlBody, textBody)

	// L'enveloppe SMTP (MAIL FROM) exige une adresse nue. Un éventuel nom
	// d'affichage ("Breezy <no-reply@…>") ne va que dans l'en-tête From.
	envelopeFrom := m.from
	if parsed, err := mail.ParseAddress(m.from); err == nil {
		envelopeFrom = parsed.Address
	}

	addr := net.JoinHostPort(m.host, m.port)
	conn, err := net.DialTimeout("tcp", addr, smtpDialTimeout)
	if err != nil {
		return fmt.Errorf("dial smtp %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS si le serveur le propose (Mailpit en dev ne le propose pas).
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	// Auth uniquement si un username est configuré (Mailpit accepte sans auth).
	if m.username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(envelopeFrom); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}
