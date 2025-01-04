package mailer

import (
	"context"
	"time"

	"github.com/a-h/templ"
	"github.com/wneessen/go-mail"
)

const dialAndSendTries = 3

type Mailer struct {
	dialer *mail.Client
	sender string
}

func New(
	host string,
	port int,
	username, password, sender string,
	env string,
) Mailer {
	var dialer *mail.Client
	switch env {
	case "development":
		c, err := mail.NewClient(
			host,
			mail.WithPort(port),
			mail.WithUsername(username),
			mail.WithPassword(password),
			mail.WithTimeout(5*time.Second),
			mail.WithTLSPolicy(mail.NoTLS),
		)
		if err != nil {
			panic(err)
		}
		dialer = c
	default:
		c, err := mail.NewClient(
			host,
			mail.WithTLSPortPolicy(mail.TLSMandatory),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(username),
			mail.WithPassword(password),
		)
		if err != nil {
			panic(err)
		}
		dialer = c
	}
	return Mailer{dialer: dialer, sender: sender}
}

func (m Mailer) Send(
	recipient string,
	subjectTemplate templ.Component,
	plainBodyTemplate templ.Component,
	htmlTemplate templ.Component,
) error {
	msg := mail.NewMsg()

	if err := msg.From(m.sender); err != nil {
		return err
	}

	if err := msg.To(recipient); err != nil {
		return err
	}

	// Template for subject
	subject := templ.GetBuffer()
	defer templ.ReleaseBuffer(subject)
	if err := subjectTemplate.Render(
		context.Background(),
		subject,
	); err != nil {
		return err
	}
	msg.Subject(subject.String())

	// Template for plain text
	plainBody := templ.GetBuffer()
	defer templ.ReleaseBuffer(plainBody)
	if err := plainBodyTemplate.Render(
		context.Background(),
		subject,
	); err != nil {
		return err
	}
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())

	// Template for html
	html := templ.GetBuffer()
	defer templ.ReleaseBuffer(html)
	if err := htmlTemplate.Render(context.Background(), html); err != nil {
		return err
	}
	msg.AddAlternativeString(mail.TypeTextHTML, html.String())

	// Sending the message
	//
	// Trying 3 times with half a second delay in case something goes wrong
	for i := 1; i <= dialAndSendTries; i++ {
		if err := m.dialer.DialAndSend(msg); err != nil {
			if i == dialAndSendTries {
				return err
			}
		} else {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}
