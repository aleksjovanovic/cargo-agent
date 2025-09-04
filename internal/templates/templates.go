// internal/templates/templates.go
package templates

import (
	"bytes"
	htmltmpl "html/template"
	"sync"
	texttmpl "text/template"
	"time"
)

// We keep compiled templates in package-level vars for fast reuse.
// html/template is used for HTML bodies (auto-escaping), while text/template
// is used for the plain-text alternatives.
var (
	verifyHTMLTpl *htmltmpl.Template
	verifyTextTpl *texttmpl.Template

	resetHTMLTpl *htmltmpl.Template
	resetTextTpl *texttmpl.Template

	initOnce sync.Once
)

// MustInit parses and compiles all inline templates exactly once.
// It panics on error because a broken template means the process cannot
// reliably send emails.
func MustInit() {
	initOnce.Do(func() {
		var err error

		// Verification email templates
		verifyHTMLTpl, err = htmltmpl.New("verify_html").Parse(htmlEmailTmpl)
		if err != nil {
			panic(err)
		}
		verifyTextTpl, err = texttmpl.New("verify_txt").Parse(textEmailTmpl)
		if err != nil {
			panic(err)
		}

		// Password reset templates
		resetHTMLTpl, err = htmltmpl.New("reset_html").Parse(resetHtmlEmailTmpl)
		if err != nil {
			panic(err)
		}
		resetTextTpl, err = texttmpl.New("reset_txt").Parse(resetTextEmailTmpl)
		if err != nil {
			panic(err)
		}
	})
}

// Inline HTML template for email verification.
const htmlEmailTmpl = `
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Verify your account</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body style="margin:0;padding:0;background:#f6f8fb;">
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f6f8fb;padding:24px 0;">
      <tr>
        <td align="center">
          <table role="presentation" width="600" cellspacing="0" cellpadding="0" style="max-width:600px;background:#ffffff;border-radius:12px;overflow:hidden;font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,'Helvetica Neue',Arial,sans-serif;">
            <tr>
              <td style="padding:24px 28px;border-bottom:1px solid #eef2f7;">
                <h1 style="margin:0;font-size:20px;font-weight:700;color:#0f172a;">Cargo Agent</h1>
              </td>
            </tr>
            <tr>
              <td style="padding:28px;">
                <h2 style="margin:0 0 12px 0;font-size:18px;color:#0f172a;">Verify your email</h2>
                <p style="margin:0 0 16px 0;font-size:14px;line-height:1.6;color:#334155;">
                  Hi {{.Username}}, please confirm your email to activate your account.
                </p>
                <p style="margin:0 0 16px 0;font-size:14px;line-height:1.6;color:#334155;">
                  This link expires on <strong>{{.ValidUntil}}</strong>.
                </p>

                <p style="margin:24px 0;">
                  <a href="{{.Link}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;font-weight:600;padding:12px 18px;border-radius:8px;">
                    Verify email
                  </a>
                </p>

                <p style="margin:16px 0 0 0;font-size:12px;line-height:1.6;color:#64748b;">
                  If the button doesn’t work, copy and paste this URL into your browser:<br />
                  <span style="word-break:break-all;color:#1e293b;">{{.Link}}</span>
                </p>
              </td>
            </tr>
            <tr>
              <td style="padding:16px 28px;background:#f8fafc;border-top:1px solid #eef2f7;font-size:12px;color:#94a3b8;">
                © {{printf "%d" .Year}} Cargo Agent
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>
`

// Inline plain-text template for email verification.
const textEmailTmpl = `Verify your email

Hi {{.Username}}, please confirm your email to activate your account.
This link expires on {{.ValidUntil}}.

Verify: {{.Link}}
`

// Inline HTML template for password reset.
const resetHtmlEmailTmpl = `
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Reset your password</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body style="margin:0;padding:0;background:#f6f8fb;">
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f6f8fb;padding:24px 0;">
      <tr>
        <td align="center">
          <table role="presentation" width="600" cellspacing="0" cellpadding="0" style="max-width:600px;background:#ffffff;border-radius:12px;overflow:hidden;font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,'Helvetica Neue',Arial,sans-serif;">
            <tr>
              <td style="padding:24px 28px;border-bottom:1px solid #eef2f7;">
                <h1 style="margin:0;font-size:20px;font-weight:700;color:#0f172a;">Cargo Agent</h1>
              </td>
            </tr>
            <tr>
              <td style="padding:28px;">
                <h2 style="margin:0 0 12px 0;font-size:18px;color:#0f172a;">Reset your password</h2>
                <p style="margin:0 0 16px 0;font-size:14px;line-height:1.6;color:#334155;">
                  Hi {{.Username}}, we received a request to reset your password.
                </p>
                <p style="margin:0 0 16px 0;font-size:14px;line-height:1.6;color:#334155;">
                  This link expires on <strong>{{.ValidUntil}}</strong>. If you didn’t request this change, you can safely ignore this email.
                </p>

                <p style="margin:24px 0;">
                  <a href="{{.Link}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;font-weight:600;padding:12px 18px;border-radius:8px;">
                    Reset password
                  </a>
                </p>

                <p style="margin:16px 0 0 0;font-size:12px;line-height:1.6;color:#64748b;">
                  If the button doesn’t work, copy and paste this URL into your browser:<br />
                  <span style="word-break:break-all;color:#1e293b;">{{.Link}}</span>
                </p>
              </td>
            </tr>
            <tr>
              <td style="padding:16px 28px;background:#f8fafc;border-top:1px solid #eef2f7;font-size:12px;color:#94a3b8;">
                © {{printf "%d" .Year}} Cargo Agent
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>
`

// Inline plain-text template for password reset.
const resetTextEmailTmpl = `Reset your password

Hi {{.Username}}, we received a request to reset your password.
This link expires on {{.ValidUntil}}.

Reset: {{.Link}}
`

// RenderVerificationEmail returns the subject, HTML body and plain-text body
// for an email verification message. It lazily initializes templates if
// MustInit hasn't been called yet.
func RenderVerificationEmail(username, link string, validUntil time.Time) (string, string, string, error) {
	// Lazy init in case the caller forgot to call MustInit at startup.
	if verifyHTMLTpl == nil || verifyTextTpl == nil {
		MustInit()
	}

	data := struct {
		Username   string
		Link       string
		ValidUntil string
		Year       int
	}{
		Username:   username,
		Link:       link,
		ValidUntil: validUntil.Format(time.RFC1123),
		Year:       time.Now().Year(),
	}

	var hb bytes.Buffer
	if err := verifyHTMLTpl.Execute(&hb, data); err != nil {
		return "", "", "", err
	}

	var tb bytes.Buffer
	if err := verifyTextTpl.Execute(&tb, data); err != nil {
		return "", "", "", err
	}

	subject := "Verify your Cargo Agent account"
	return subject, hb.String(), tb.String(), nil
}

// RenderPasswordResetEmail returns the subject, HTML body and plain-text body
// for a password reset message. It lazily initializes templates if needed.
func RenderPasswordResetEmail(username, link string, validUntil time.Time) (subject, htmlBody, textBody string, err error) {
	// Lazy init in case the caller forgot to call MustInit at startup.
	if resetHTMLTpl == nil || resetTextTpl == nil {
		MustInit()
	}

	data := struct {
		Username   string
		Link       string
		ValidUntil string
		Year       int
	}{
		Username:   username,
		Link:       link,
		ValidUntil: validUntil.Format(time.RFC1123),
		Year:       time.Now().Year(),
	}

	var hb, tb bytes.Buffer
	if err = resetHTMLTpl.Execute(&hb, data); err != nil {
		return "", "", "", err
	}
	if err = resetTextTpl.Execute(&tb, data); err != nil {
		return "", "", "", err
	}

	subject = "Reset your password"
	htmlBody = hb.String()
	textBody = tb.String()
	return
}
