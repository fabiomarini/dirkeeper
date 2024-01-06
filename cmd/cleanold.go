package cmd

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/fs"
	"os"
	"path"
	"time"
)

func init() {
	CleanOldCmd.PersistentFlags().StringSliceVarP(&cleanOldParams.dirNames, "directory", "d", []string{}, "List of directories to cleanup")
	CleanOldCmd.PersistentFlags().IntVar(&cleanOldParams.maxAgeDays, "max-age", 0, "Maximum age of the file in days")
	CleanOldCmd.PersistentFlags().BoolVar(&cleanOldParams.dryRun, "dry-run", false, "Only check for old files without deleting")
}

type CleanOldParamsType struct {
	dirNames   []string
	maxAgeDays int
	dryRun     bool
}

var cleanOldParams = CleanOldParamsType{}

var CleanOldCmd = &cobra.Command{
	Use:   "cleanold",
	Short: "clean old files",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanOldFiles(cleanOldParams)
	},
}

func cleanOldFiles(params CleanOldParamsType) error {
	if err := checkParameters(params); err != nil {
		return err
	}

	maxAgeDays := params.maxAgeDays
	dryRun := params.dryRun

	for _, dirName := range params.dirNames {
		if err := cleanupDirectory(dirName, maxAgeDays, dryRun); err != nil {
			log.Errorln("Error cleaning directory", dirName, err.Error())
			return err
		}
		log.Infoln("Directory", dirName, "cleaned")
	}
	return nil
}

func cleanupDirectory(dirName string, maxAgeDays int, dryRun bool) error {
	directory, err := os.Open(dirName)
	if err != nil {
		return err
	}
	defer func(directory *os.File) {
		err := directory.Close()
		if err != nil {
			log.Warnln("Error closing directory", dirName, err.Error())
		}
	}(directory)

	dirContent, err := directory.Readdir(-1)
	if err != nil {
		return err
	}

	startDate := time.Now().AddDate(0, 0, -maxAgeDays)
	log.Infof("Cleaning directory %v from files created before %v (%d days old)", dirName, startDate.Format("2006-01-02"), maxAgeDays)
	for _, fileInfo := range dirContent {
		if err := checkIfFileIsOld(dirName, fileInfo, startDate, dryRun); err != nil {
			return err
		}
	}
	return nil
}

func checkIfFileIsOld(dirName string, fileInfo os.FileInfo, startDate time.Time, dryRun bool) error {
	fileName := fileInfo.Name()
	fileModTime := fileInfo.ModTime()
	if fileModTime.Before(startDate) {
		if fileInfo.IsDir() {
			log.Infoln("Skipping directory", fileName)
			return nil
		}
		if fileInfo.Mode() == fs.ModeSymlink {
			log.Infoln("Skipping symlink", fileName)
			return nil
		}

		fileAge := time.Now().Sub(fileModTime).Hours() / 24
		if dryRun {
			fn := fileName
			if len(fileName) > 30 {
				fn = fileName[:27] + "..."
			}
			log.Infof("Candidate file %-30v\t%10d bytes\t%v\t%.0f days old", fn, fileInfo.Size(), fileModTime.Format(time.RFC3339), fileAge)
		} else {
			log.Infof("Deleting %-30v\t%10d bytes\t%v\t%.0f days old", fileName, fileInfo.Size(), fileModTime.Format(time.RFC3339), fileAge)
			if err := os.Remove(path.Join(dirName, fileName)); err != nil {
				log.Errorln("Impossible to delete file", fileName)
				return err
			}
		}
	}
	return nil
}

func checkParameters(params CleanOldParamsType) error {
	if len(params.dirNames) <= 0 {
		log.Errorln("Missing directory param")
		return errors.New("missing directory")
	}

	for _, dirName := range params.dirNames {
		directory, err := os.Open(dirName)
		if err != nil {
			log.Errorln("Invalid directory", err)
			return err
		}

		fileIn, err := directory.Stat()
		if err != nil {
			log.Errorln("Error reading directory info", err)
			return err
		}

		if !fileIn.IsDir() {
			log.Errorln("Directory argument must be a valid directory")
			return errors.New("invalid directory")
		}

	}
	if params.maxAgeDays <= 0 {
		log.Errorln("Invalid maxAgeDays, positive number expected")
		return errors.New("invalid maxAgeDays")
	}

	return nil
}
