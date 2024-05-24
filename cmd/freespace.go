package cmd

import (
	"bytes"
	"dirkeeper/internal/utils"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"os"
	"syscall"
	"text/template"
)

type freeSpaceCmdParamsType struct {
	utils.EmailParams
	Path            string
	LimitPercentage float64
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
	FreeSpaceCmd.Flags().BoolVar(&freeSpaceCmdParams.Quiet, "quiet", false, "Do not print notification")
	utils.MapFlags(FreeSpaceCmd.Flags(), &freeSpaceCmdParams.EmailParams)
}

var freeSpaceCmdParams = freeSpaceCmdParamsType{
	Path:            "/",
	LimitPercentage: 15,
	EmailParams: utils.EmailParams{
		EmailEnabled: false,
		EmailTo:      []string{""},
		SMTPServer:   "",
		SMTPPort:     25,
		SMTPUser:     "",
		SMTPPassword: "",
		SMTPFrom:     "",
		SMTPTLS:      false,
		SMTPAuthType: "plain",
		SMTPSubject:  "",
	},
	Quiet: false,
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
	if !params.EmailEnabled {
		return nil
	}
	if err := utils.CheckEmailParams(params.EmailParams); err != nil {
		return err
	}

	subject := "Free space notification"
	if params.SMTPSubject != "" {
		subject = params.SMTPSubject
	}
	body, err := buildMailBody(freeSpaceInfo)
	if err != nil {
		return fmt.Errorf("failed to build mail body: %s", err)
	}

	if !params.Quiet {
		fmt.Printf("Sending notification email to %v\nInfo: %#v\n", params.EmailTo, freeSpaceInfo)
	}
	return utils.SendEmail(subject, body, params.EmailParams)
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
