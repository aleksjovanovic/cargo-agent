// internal/templates/templates.go
package templates

import (
	"bytes"
	"text/template"
	txttmpl "text/template"
	"time"
)

var (
	htmlTpl *template.Template
	textTpl *txttmpl.Template
)

// Pozovi iz main() jednom pri startu (nije skroz nužno, ali je jasno).
func MustInit() {
	var err error
	htmlTpl, err = template.New("verify_html").Parse(htmlEmailTmpl)
	if err != nil {
		panic(err)
	}
	textTpl, err = txttmpl.New("verify_txt").Parse(textEmailTmpl)
	if err != nil {
		panic(err)
	}
}

type verifyData struct {
	Username   string
	Link       string
	ValidUntil string
}

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

const textEmailTmpl = `Verify your email

Hi {{.Username}}, please confirm your email to activate your account.
This link expires on {{.ValidUntil}}.

Verify: {{.Link}}
`

// RenderVerificationEmail vraća (subject, htmlBody, textBody).
func RenderVerificationEmail(username, link string, validUntil time.Time) (string, string, string, error) {
	// lenja inicijalizacija ako MustInit nije pozvan
	if htmlTpl == nil || textTpl == nil {
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
	if err := htmlTpl.Execute(&hb, data); err != nil {
		return "", "", "", err
	}

	var tb bytes.Buffer
	if err := textTpl.Execute(&tb, data); err != nil {
		return "", "", "", err
	}

	subject := "Verify your Cargo Agent account"
	return subject, hb.String(), tb.String(), nil
}
