package utils

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/wneessen/go-mail"
)

type EmailParams struct {
	EmailEnabled bool
	EmailTo      []string
	SMTPServer   string
	SMTPPort     int16
	SMTPAuthType string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPTLS      bool
	SMTPSubject  string
}

func SendEmail(subject, body string, params EmailParams) error {
	m := mail.NewMsg()
	if err := m.From(params.SMTPFrom); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	if err := m.To(params.EmailTo...); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}
	m.Subject(subject)
	m.SetBodyString(mail.TypeTextPlain, body)

	options := make([]mail.Option, 1)
	options = append(options, mail.WithPort(int(params.SMTPPort)))
	if !params.SMTPTLS {
		options = append(options, mail.WithTLSPortPolicy(mail.NoTLS))
	} else {
		options = append(options, mail.WithTLSPortPolicy(mail.TLSOpportunistic))
	}

	if params.SMTPUser != "" && params.SMTPPassword != "" {
		switch params.SMTPAuthType {
		case "login":
			options = append(options, mail.WithSMTPAuth(mail.SMTPAuthLogin))
		case "oauth":
			options = append(options, mail.WithSMTPAuth(mail.SMTPAuthXOAUTH2))
		case "plain":
			options = append(options, mail.WithSMTPAuth(mail.SMTPAuthPlain))
		}

		options = append(options,
			mail.WithUsername(params.SMTPUser),
			mail.WithPassword(params.SMTPPassword),
		)
	}

	c, err := mail.NewClient(params.SMTPServer, options...)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %s", err)
	}
	if err := c.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}
	return nil
}

func CheckEmailParams(params EmailParams) error {
	if !params.EmailEnabled {
		return nil
	}
	if params.SMTPServer == "" {
		return fmt.Errorf("missing SMTP server")
	}
	if params.SMTPPort == 0 {
		return fmt.Errorf("missing SMTP port")
	}
	if params.SMTPFrom == "" {
		return fmt.Errorf("missing SMTP from")
	}
	if params.SMTPAuthType != "" {
		switch params.SMTPAuthType {
		case "plain", "oauth", "login":
		default:
			return fmt.Errorf("wrong SMTP auth type")
		}
	}
	if params.EmailTo == nil || len(params.EmailTo) == 0 {
		return fmt.Errorf("missing email to")
	}
	return nil
}

func MapFlags(flags *pflag.FlagSet, params *EmailParams) {
	flags.BoolVar(&params.EmailEnabled, "email", false, "Send email notification")
	flags.StringSliceVar(&params.EmailTo, "email-to", []string{}, "Email address to send notification")
	flags.StringVar(&params.SMTPServer, "smtp-server", "", "SMTP server")
	flags.Int16Var(&params.SMTPPort, "smtp-port", 25, "SMTP port")
	flags.StringVar(&params.SMTPAuthType, "smtp-auth-type", "", "SMTP auth type (plain, oauth)")
	flags.StringVar(&params.SMTPUser, "smtp-user", "", "SMTP user")
	flags.StringVar(&params.SMTPPassword, "smtp-password", "", "SMTP password")
	flags.StringVar(&params.SMTPFrom, "smtp-from", "", "SMTP from")
	flags.BoolVar(&params.SMTPTLS, "smtp-tls", false, "Use TLS")
	flags.StringVar(&params.SMTPSubject, "smtp-subject", "", "SMTP subject")
}
