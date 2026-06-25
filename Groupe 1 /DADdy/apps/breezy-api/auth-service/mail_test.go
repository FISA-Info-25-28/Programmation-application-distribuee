package main

import (
	"strings"
	"testing"
)

func TestNewMailer(t *testing.T) {
	t.Parallel()
	if _, ok := newMailer(authConfig{}).(logMailer); !ok {
		t.Error("empty SMTP_HOST should yield a logMailer")
	}
	if _, ok := newMailer(authConfig{SMTPHost: "smtp.example.com"}).(smtpMailer); !ok {
		t.Error("configured SMTP_HOST should yield an smtpMailer")
	}
}

func TestLogMailerNeverErrors(t *testing.T) {
	t.Parallel()
	m := logMailer{}
	if err := m.SendVerificationEmail("a@b.c", "https://x/y"); err != nil {
		t.Errorf("SendVerificationEmail: %v", err)
	}
	if err := m.SendPasswordResetEmail("a@b.c", "https://x/y"); err != nil {
		t.Errorf("SendPasswordResetEmail: %v", err)
	}
	if err := m.SendPasswordChangedEmail("a@b.c"); err != nil {
		t.Errorf("SendPasswordChangedEmail: %v", err)
	}
}

func TestBuildMIMEMessage(t *testing.T) {
	t.Parallel()
	msg := string(buildMIMEMessage("from@x", "to@y", "Sujet accentué é", "<p>html</p>", "texte"))

	for _, want := range []string{
		"From: from@x\r\n",
		"To: to@y\r\n",
		"MIME-Version: 1.0\r\n",
		"multipart/alternative; boundary=",
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n",
		"Content-Type: text/html; charset=\"UTF-8\"\r\n",
		"Content-Transfer-Encoding: quoted-printable\r\n",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("MIME message missing %q\n--- got ---\n%s", want, msg)
		}
	}
	// Sujet non-ASCII => en-tête encodé RFC 2047.
	if !strings.Contains(msg, "=?UTF-8?") {
		t.Error("non-ASCII subject should be MIME-encoded")
	}
	// La partie texte doit précéder la partie HTML (clients affichent la dernière rendue).
	if strings.Index(msg, "text/plain") > strings.Index(msg, "text/html") {
		t.Error("text part should come before html part")
	}
}

func TestEncodeQuotedPrintable(t *testing.T) {
	t.Parallel()
	// Un caractère non-ASCII doit être encodé (=XX) et non laissé brut.
	out := encodeQuotedPrintable("café")
	if strings.Contains(out, "é") || !strings.Contains(out, "=") {
		t.Errorf("expected quoted-printable encoding, got %q", out)
	}
	// ASCII pur reste lisible.
	if got := encodeQuotedPrintable("abcdef"); got != "abcdef" {
		t.Errorf("ascii should pass through, got %q", got)
	}
}

func TestEncodeHeaderWord(t *testing.T) {
	t.Parallel()
	if got := encodeHeaderWord("plain ascii"); got != "plain ascii" {
		t.Errorf("ascii header should be unchanged, got %q", got)
	}
	if got := encodeHeaderWord("accentué"); !strings.HasPrefix(got, "=?UTF-8?") {
		t.Errorf("non-ascii header should be encoded, got %q", got)
	}
}

func TestEmailTemplatesContainLink(t *testing.T) {
	t.Parallel()
	const link = "https://breezy.example/verify?token=abc"

	tests := []struct {
		name     string
		body     string
		wantLink bool
	}{
		{"verification text", verificationEmailText(link), true},
		{"verification html", verificationEmailHTML(link), true},
		{"reset text", passwordResetEmailText(link), true},
		{"reset html", passwordResetEmailHTML(link), true},
		{"changed text", passwordChangedEmailText(), false},
		{"changed html", passwordChangedEmailHTML(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if strings.TrimSpace(tt.body) == "" {
				t.Fatal("body is empty")
			}
			if tt.wantLink && !strings.Contains(tt.body, link) {
				t.Errorf("body should contain the link %q", link)
			}
			if !tt.wantLink && strings.Contains(tt.body, link) {
				t.Error("security alert email must not embed an action link")
			}
		})
	}
}

func TestEmailHTMLEscapesLink(t *testing.T) {
	t.Parallel()
	// Un lien contenant des métacaractères HTML doit être échappé dans le HTML.
	out := verificationEmailHTML("https://x/y?a=1&b=2")
	if strings.Contains(out, "a=1&b=2") {
		t.Error("ampersand in link should be HTML-escaped")
	}
	if !strings.Contains(out, "a=1&amp;b=2") {
		t.Error("expected &amp; in escaped link")
	}
}

func TestEmailShellStructure(t *testing.T) {
	t.Parallel()
	out := emailShell("<p>inner</p>")
	for _, want := range []string{"<!DOCTYPE html>", "<p>inner</p>", "</html>"} {
		if !strings.Contains(out, want) {
			t.Errorf("emailShell missing %q", want)
		}
	}
}

func TestEmailButtonAndFallback(t *testing.T) {
	t.Parallel()
	btn := emailButton("Click", "https://x/y")
	if !strings.Contains(btn, "https://x/y") || !strings.Contains(btn, "Click") {
		t.Errorf("button missing label or link: %q", btn)
	}
	fb := emailFallbackLink("https://x/y")
	if strings.Count(fb, "https://x/y") < 2 {
		t.Errorf("fallback should show the link as href and text: %q", fb)
	}
}
