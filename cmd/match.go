package cmd

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

func init() {
	MatchCmd.PersistentFlags().StringVarP(&matchCmdParams.dirName, "directory", "d", "", "Base directory")
	MatchCmd.PersistentFlags().StringVar(&matchCmdParams.destDir, "dest-dir", "", "Destination directory")
	MatchCmd.PersistentFlags().StringVarP(&matchCmdParams.action, "action", "a", "", "Action to execute (copy, copy-delete, move, delete")
	MatchCmd.PersistentFlags().StringSliceVar(&matchCmdParams.prefixes, "prefix", []string{}, "List of file name prefixes")
	MatchCmd.PersistentFlags().StringSliceVar(&matchCmdParams.suffixes, "suffix", []string{}, "List of file name suffixes")
	MatchCmd.PersistentFlags().StringSliceVar(&matchCmdParams.patterns, "pattern", []string{}, "List of file name patterns")
	MatchCmd.PersistentFlags().BoolVar(&matchCmdParams.dryRun, "dry-run", false, "Do not execute action")
	MatchCmd.PersistentFlags().IntVar(&matchCmdParams.maxAge, "max-age", 0, "Max file age in minutes")
}

type matchCmdParamsType struct {
	dirName  string
	destDir  string
	action   string
	prefixes []string
	suffixes []string
	patterns []string
	maxAge   int
	dryRun   bool
}

var matchCmdParams = matchCmdParamsType{}

var MatchCmd = &cobra.Command{
	Use:   "match",
	Short: "match and process files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return MatchFiles(matchCmdParams)
	},
}

func MatchFiles(params matchCmdParamsType) error {
	if err := checkMatchParameters(params); err != nil {
		return err
	}

	directory, err := os.Open(params.dirName)
	if err != nil {
		return err
	}
	defer func(directory *os.File) {
		err := directory.Close()
		if err != nil {
			log.Warnln("Error closing directory", params.dirName, err.Error())
		}
	}(directory)

	dirContent, err := directory.Readdir(-1)
	if err != nil {
		return err
	}

	var patterns = make([]*regexp.Regexp, len(params.patterns))
	for i, p := range params.patterns {
		re, _ := regexp.Compile(p)
		patterns[i] = re
	}

	log.Infof("Scanning directory %v for matches", params.dirName)
	for _, fileInfo := range dirContent {
		if err := checkAndProcessFile(params, fileInfo, patterns); err != nil {
			return err
		}
	}
	log.Infoln("Directory", params.dirName, "scan complete")
	return nil
}

func checkAndProcessFile(params matchCmdParamsType, fileInfo os.FileInfo, patterns []*regexp.Regexp) error {
	fileName := fileInfo.Name()
	if fileInfo.IsDir() {
		log.Infoln("Skipping directory", fileName)
		return nil
	}
	if fileInfo.Mode() == fs.ModeSymlink {
		log.Infoln("Skipping symlink", fileName)
		return nil
	}

	if params.maxAge > 0 && int(time.Now().Sub(fileInfo.ModTime()).Minutes()) > params.maxAge {
		return nil
	}

	for _, prefix := range params.prefixes {
		if strings.HasPrefix(fileName, prefix) {
			log.Infoln("File", fileName, "matches prefix", prefix)
			if !params.dryRun {
				if err := processFile(params.action, params.dirName, params.destDir, fileName); err != nil {
					log.Errorf("Error processing file %v: %v", fileName, err.Error())
				}
			}
		}
	}
	for _, suffix := range params.suffixes {
		if strings.HasSuffix(fileName, suffix) {
			log.Infoln("File", fileName, "matches suffix", suffix)
			if !params.dryRun {
				if err := processFile(params.action, params.dirName, params.destDir, fileName); err != nil {
					log.Errorf("Error processing file %v: %v", fileName, err.Error())
				}
			}
		}
	}
	for _, pattern := range patterns {
		if pattern.MatchString(fileName) {
			log.Infoln("File", fileName, "matches pattern", pattern)
			if !params.dryRun {
				if err := processFile(params.action, params.dirName, params.destDir, fileName); err != nil {
					log.Errorf("Error processing file %v: %v", fileName, err.Error())
				}
			}
		}
	}
	return nil
}

func checkMatchParameters(params matchCmdParamsType) error {
	directory, err := os.Open(params.dirName)
	if err != nil {
		log.Errorln("Invalid directory", err)
		return err
	}
	defer func(directory *os.File) {
		err := directory.Close()
		if err != nil {
			log.Warnln("Error closing directory", params.dirName, err.Error())
		}
	}(directory)

	fileIn, err := directory.Stat()
	if err != nil {
		log.Errorln("Error reading directory info", err)
		return err
	}

	if !fileIn.IsDir() {
		log.Errorln("Directory argument must be a valid directory")
		return errors.New("invalid directory")
	}

	if params.maxAge < 0 {
		log.Errorln("Max Age cannot must be greather than zero")
		return errors.New("invalid max-age")
	}

	switch strings.ToUpper(params.action) {
	case "COPY", "MOVE", "COPY-DELETE":
		if len(params.destDir) == 0 {
			log.Errorln("Missing destination directory")
			return errors.New("missing destination directory")
		}
		destDir, err := os.Open(params.destDir)
		if err != nil {
			log.Errorln("Invalid directory", err)
			return err
		}
		if err := destDir.Close(); err != nil {
			log.Warnln("Error closing destination directory", err.Error())
		}

	case "DELETE":
	default:
		log.Errorln("Invalid action", params.action)
		return errors.New("invalid action")
	}

	if len(params.prefixes) == 0 && len(params.suffixes) == 0 && len(params.patterns) == 0 {
		log.Errorln("At least one fo prefix, suffix or pattern must be specified")
		return errors.New("no matcher specified")
	}

	if len(params.patterns) > 0 {
		for _, p := range params.patterns {
			if _, err := regexp.Compile(p); err != nil {
				log.Errorln("Invalid pattern regexp", p, err.Error())
				return err
			}
		}
	}

	return nil
}

func processFile(action, sourceDir, destDir, fileName string) error {
	switch strings.ToUpper(action) {
	case "COPY":
		log.Infof("Copying file %v to directory %v", fileName, destDir)
		if err := copyFile(path.Join(sourceDir, fileName), path.Join(destDir, fileName)); err != nil {
			log.Errorf("Error copying file %v: %v", fileName, err.Error())
			return err
		}
	case "COPY-DELETE":
		log.Infof("Copying file %v to directory %v", fileName, destDir)
		if err := copyFile(path.Join(sourceDir, fileName), path.Join(destDir, fileName)); err != nil {
			log.Errorf("Error copying file %v: %v", fileName, err.Error())
			return err
		}
		log.Infof("Deleting file %v", fileName)
		if err := deleteFile(path.Join(sourceDir, fileName)); err != nil {
			log.Errorf("Error deleting file %v: %v", fileName, err.Error())
			return err
		}
	case "MOVE":
		log.Infof("Moving file %v to directory %v", fileName, destDir)
		if err := moveFile(path.Join(sourceDir, fileName), path.Join(destDir, fileName)); err != nil {
			log.Errorf("Error moving file %v: %v", fileName, err.Error())
			return err
		}
	case "DELETE":
		log.Infof("Deleting file %v", fileName)
		if err := deleteFile(path.Join(sourceDir, fileName)); err != nil {
			log.Errorf("Error deleting file %v: %v", fileName, err.Error())
			return err
		}
	}
	return nil
}

func copyFile(fromFile string, toFile string) error {
	from, err := os.Open(fromFile)
	if err != nil {
		return err
	}
	defer func(from *os.File) {
		err := from.Close()
		if err != nil {
			log.Warnln("Error closing source file", fromFile, err.Error())
		}
	}(from)

	to, err := os.Create(toFile)
	if err != nil {
		return err
	}
	defer func(to *os.File) {
		err := to.Close()
		if err != nil {
			log.Warnln("Error closing destination file", toFile, err.Error())
		}
	}(to)

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	return nil
}

func moveFile(fromFile string, toFile string) error {
	return os.Rename(fromFile, toFile)
}

func deleteFile(fileName string) error {
	return os.Remove(fileName)
}
