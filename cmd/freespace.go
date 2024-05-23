package cmd

import (
	"bytes"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/wneessen/go-mail"
	"os"
	"syscall"
	"text/template"
)

type freeSpaceCmdParamsType struct {
	Path            string
	LimitPercentage float64
	Email           bool
	EmailTo         []string
	SMTPServer      string
	SMTPPort        int16
	SMTPUser        string
	SMTPPassword    string
	SMTPFrom        string
	SMTPTLS         bool
	SMTPSubject     string
	Quiet           bool
}

type freeSpaceInfoType struct {
	Hostname        string
	Path            string
	TotalSize       uint64
	FreeSize        uint64
	FreePercentage  float64
	LimitPercentage float64
}

var FreeSpaceCmd = &cobra.Command{
	Use:   "freespace",
	Short: "check free disk space",
	RunE: func(cmd *cobra.Command, args []string) error {
		freeSpace, err := checkFreeSpace(freeSpaceCmdParams)
		if err != nil {
			return notifyFreeSpaceError(freeSpace, freeSpaceCmdParams)
		}
		return nil
	},
}

func init() {
	FreeSpaceCmd.Flags().StringVarP(&freeSpaceCmdParams.Path, "path", "p", "/", "Path to check")
	FreeSpaceCmd.Flags().Float64Var(&freeSpaceCmdParams.LimitPercentage, "limit", 15, "Limit percentage")
	FreeSpaceCmd.Flags().BoolVar(&freeSpaceCmdParams.Email, "email", false, "Send email notification")
	FreeSpaceCmd.Flags().StringSliceVar(&freeSpaceCmdParams.EmailTo, "email-to", []string{}, "Email address to send notification")
	FreeSpaceCmd.Flags().StringVar(&freeSpaceCmdParams.SMTPServer, "smtp-server", "", "SMTP server")
	FreeSpaceCmd.Flags().Int16Var(&freeSpaceCmdParams.SMTPPort, "smtp-port", 25, "SMTP port")
	FreeSpaceCmd.Flags().StringVar(&freeSpaceCmdParams.SMTPUser, "smtp-user", "", "SMTP user")
	FreeSpaceCmd.Flags().StringVar(&freeSpaceCmdParams.SMTPPassword, "smtp-password", "", "SMTP password")
	FreeSpaceCmd.Flags().StringVar(&freeSpaceCmdParams.SMTPFrom, "smtp-from", "", "SMTP from")
	FreeSpaceCmd.Flags().BoolVar(&freeSpaceCmdParams.SMTPTLS, "smtp-tls", false, "Use TLS")
	FreeSpaceCmd.Flags().StringVar(&freeSpaceCmdParams.SMTPSubject, "smtp-subject", "", "SMTP subject")
	FreeSpaceCmd.Flags().BoolVar(&freeSpaceCmdParams.Quiet, "quiet", false, "Do not print notification")
}

var freeSpaceCmdParams = freeSpaceCmdParamsType{
	Path:            "/",
	LimitPercentage: 15,
	Email:           false,
	EmailTo:         []string{""},
	SMTPServer:      "",
	SMTPPort:        25,
	SMTPUser:        "",
	SMTPPassword:    "",
	SMTPFrom:        "",
	SMTPTLS:         false,
	SMTPSubject:     "",
	Quiet:           false,
}

func checkFreeSpace(params freeSpaceCmdParamsType) (freeSpaceInfoType, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(params.Path, &fs)
	if err != nil {
		return freeSpaceInfoType{}, err
	}
	totalSpace := fs.Blocks * uint64(fs.Bsize)
	freeSpace := fs.Bavail * uint64(fs.Bsize)

	threshold := float64(totalSpace) * (params.LimitPercentage / 100.0)
	freeSpacePercent := float64(freeSpace) / float64(totalSpace)

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		hostname = "<unknown>"
	}

	freeSpaceInfo := freeSpaceInfoType{
		Hostname:        hostname,
		Path:            params.Path,
		TotalSize:       totalSpace,
		FreeSize:        freeSpace,
		FreePercentage:  freeSpacePercent * 100.0,
		LimitPercentage: params.LimitPercentage,
	}

	if float64(freeSpace) < threshold {
		return freeSpaceInfo, fmt.Errorf("not enough free space on server %v - path: %v: %.2f %% of total", hostname, params.Path, freeSpacePercent*100.0)
	}

	if !params.Quiet {
		fmt.Printf("Available free space on %v: %.2f %% of total (%v of %v)\n", params.Path, freeSpacePercent*100.0, humanize.Bytes(freeSpace), humanize.Bytes(totalSpace))
	}
	return freeSpaceInfo, nil
}

func notifyFreeSpaceError(freeSpaceInfo freeSpaceInfoType, params freeSpaceCmdParamsType) error {
	if !params.Email {
		return nil
	}
	if !checkEmailParams(params) {
		return fmt.Errorf("missing email notification parameters")
	}

	m := mail.NewMsg()
	if err := m.From(params.SMTPFrom); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	if err := m.To(params.EmailTo...); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}
	if params.SMTPSubject != "" {
		m.Subject(params.SMTPSubject)
	} else {
		m.Subject("Free space notification")
	}
	body, err := buildMailBody(freeSpaceInfo)
	if err != nil {
		return fmt.Errorf("failed to build mail body: %s", err)
	}
	m.SetBodyString(mail.TypeTextPlain, body)

	if !params.Quiet {
		fmt.Printf("Sending notification email to %v\nInfo: %#v\n", params.EmailTo, freeSpaceInfo)
	}

	options := make([]mail.Option, 1)
	options = append(options, mail.WithPort(int(params.SMTPPort)))
	if !params.SMTPTLS {
		options = append(options, mail.WithTLSPortPolicy(mail.NoTLS))
	}

	if params.SMTPUser != "" && params.SMTPPassword != "" {
		options = append(options,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
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

func checkEmailParams(params freeSpaceCmdParamsType) bool {
	return params.EmailTo != nil && len(params.EmailTo) > 0 && params.SMTPServer != "" && params.SMTPPort != 0 && params.SMTPFrom != ""
}

var mailTemplate = template.Must(template.New("mail").Funcs(map[string]interface{}{
	"formatBytes": humanize.Bytes,
}).Parse(`
Hostname: {{.Hostname}}
Path: {{.Path}}
Free space: {{formatBytes .FreeSize}} of {{formatBytes .TotalSize}} ({{printf "%.2f" .FreePercentage}}% of total)
`))

func buildMailBody(freeSpaceInfo freeSpaceInfoType) (string, error) {
	var bodyBuffer bytes.Buffer
	err := mailTemplate.Execute(&bodyBuffer, freeSpaceInfo)
	if err != nil {
		return "", err
	}
	return bodyBuffer.String(), nil
}
