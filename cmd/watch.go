package cmd

import (
	"errors"
	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func init() {
	WatchCmd.PersistentFlags().StringVarP(&watchCmdParams.configFile, "config", "c", "", "Config file")
	WatchCmd.Flags().BoolVar(&watchCmdParams.debug, "debug", false, "Enable debug log")
	WatchCmd.Flags().IntVar(&watchCmdParams.frequency, "frequency", 10, "Watch frequency (default 10s)")
}

type WatchCmdParamsType struct {
	configFile string
	debug      bool
	frequency  int
}
type DirWatchConfig struct {
	Name  string
	Rules []RuleConfig
}
type RuleConfig struct {
	Action      string
	Destination string
	Prefix      []string
	Pattern     []string
	Suffix      []string
}
type WatchConfig struct {
	DryRun      bool
	Directories []DirWatchConfig
}

var watchCmdParams = WatchCmdParamsType{}

var WatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "watch for new files and process them based on config rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		if watchCmdParams.debug {
			log.SetLevel(log.DebugLevel)
		}
		if len(watchCmdParams.configFile) == 0 {
			log.Errorln("Invalid config file name")
			return errors.New("invalid config file")
		}

		file, err := os.Open(watchCmdParams.configFile)
		if err != nil {
			log.Errorln("Invalid config file")
			return err

		}
		file.Close()

		config, err := initConfig(watchCmdParams.configFile)
		if err != nil {
			log.Errorln("Invalid config file content")
			return err
		}
		log.Debugf("%#v", config)

		return watch(config)
	},
}

func initConfig(configFile string) (*WatchConfig, error) {
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	//viper.Debug()
	var config struct {
		Watch WatchConfig
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	if len(config.Watch.Directories) == 0 {
		return nil, errors.New("missing directory configuration")
	}

	outConfig := WatchConfig{
		DryRun:      config.Watch.DryRun,
		Directories: make([]DirWatchConfig, len(config.Watch.Directories)),
	}

	for i, d := range config.Watch.Directories {
		dir, err := os.Open(path.Clean(d.Name))
		if err != nil {
			log.Errorln("Invalid destination directory")
			return nil, err
		}
		dirWatchConfig := &outConfig.Directories[i]
		dirWatchConfig.Name, _ = filepath.Abs(dir.Name())
		dir.Close()

		if len(d.Rules) <= 0 {
			log.Errorln("Missing rules for directory", dirWatchConfig.Name)
			return nil, errors.New("missing rules")
		}
		dirWatchConfig.Rules = make([]RuleConfig, len(d.Rules))

		for i, rule := range d.Rules {
			// Checking action
			configRule := &d.Rules[i]
			dirWatchRule := &dirWatchConfig.Rules[i]

			rule.Action = strings.ToUpper(configRule.Action)
			dirWatchRule.Action = rule.Action
			switch rule.Action {
			case "COPY", "MOVE":
				if len(configRule.Destination) == 0 {
					log.Errorln("Missing destination directory")
					return nil, errors.New("missing destination directory")
				}
				// Checking Destination
				if len(configRule.Destination) > 0 {
					file, err := os.Open(path.Clean(configRule.Destination))
					if err != nil {
						log.Errorln("Invalid destination directory")
						return nil, err
					}
					dirWatchRule.Destination, _ = filepath.Abs(file.Name())
					file.Close()
				}

			case "DELETE":
			default:
				log.Errorln("Invalid action", configRule.Action)
				return nil, errors.New("invalid action")
			}

			// Checking matcher presence
			if len(configRule.Prefix) == 0 && len(configRule.Suffix) == 0 && len(configRule.Pattern) == 0 {
				log.Errorln("At least one Prefix or Suffix or Pattern must be configured")
				return nil, errors.New("no matcher specified")
			}
			dirWatchRule.Prefix = configRule.Prefix
			dirWatchRule.Suffix = configRule.Suffix

			// Checking pattern validity
			if len(configRule.Pattern) > 0 {
				dirWatchRule.Pattern = make([]string, len(configRule.Pattern))
				for pi, p := range configRule.Pattern {
					if _, err := regexp.Compile(p); err != nil {
						log.Errorln("Invalid pattern regexp", p, err.Error())
						return nil, err
					}
					dirWatchRule.Pattern[pi] = p
				}
			}
		}
	}

	return &outConfig, nil
}

func watch(config *WatchConfig) error {
	w := watcher.New()
	w.SetMaxEvents(10)
	w.FilterOps(watcher.Create)

	go func(config *WatchConfig) {
		for {
			select {
			case event := <-w.Event:
				log.Println(event) // Print the event's info.
				if event.IsDir() {
					continue
				}

				checkFileMatch(config, event)
			case err := <-w.Error:
				log.Errorln(err)
			case <-w.Closed:
				return
			}
		}
	}(config)

	for _, dir := range config.Directories {
		// Watch this folder for changes.
		if err := w.Add(dir.Name); err != nil {
			log.Fatalln(err)
		}
	}

	// Start the watching process - it'll check for changes every 10s.
	frequency := watchCmdParams.frequency
	if frequency <= 0 {
		frequency = 10
	}
	if err := w.Start(time.Second * time.Duration(frequency)); err != nil {
		log.Fatalln(err)
	}

	doneQuitting := make(chan bool)
	go func() {
		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("Dirkeeper watcher Stopping...")

		close(doneQuitting)
	}()
	<-doneQuitting

	return nil
}

func checkFileMatch(config *WatchConfig, event watcher.Event) {
	directory, fileName := filepath.Split(event.Path)
	directory = path.Clean(directory)

	for _, dirConfig := range config.Directories {
		if dirConfig.Name != directory {
			continue
		}

		if event.IsDir() {
			log.Infoln("Skipping directory", fileName)
			return
		}
		if event.Mode() == fs.ModeSymlink {
			log.Infoln("Skipping symlink", fileName)
			return
		}

		//if params.maxAge > 0 && int(time.Now().Sub(fileInfo.ModTime()).Minutes()) > params.maxAge {
		//	continue
		//}

		for _, rule := range dirConfig.Rules {
			for _, prefix := range rule.Prefix {
				if strings.HasPrefix(fileName, prefix) {
					log.Infoln("File", fileName, "matches prefix", prefix)
					if !config.DryRun {
						if err := processFile(rule.Action, dirConfig.Name, rule.Destination, event.Name()); err != nil {
							log.Errorf("Error processing file %v: %v", fileName, err.Error())
						}
					}
				}
			}
			for _, suffix := range rule.Suffix {
				if strings.HasSuffix(fileName, suffix) {
					log.Infoln("File", fileName, "matches suffix", suffix)
					if !config.DryRun {
						if err := processFile(rule.Action, dirConfig.Name, rule.Destination, event.Name()); err != nil {
							log.Errorf("Error processing file %v: %v", fileName, err.Error())
						}
					}
				}
			}
			for _, pattern := range rule.Pattern {
				if match, _ := regexp.MatchString(pattern, fileName); match {
					log.Infoln("File", fileName, "matches pattern", pattern)
					if !config.DryRun {
						if err := processFile(rule.Action, dirConfig.Name, rule.Destination, event.Name()); err != nil {
							log.Errorf("Error processing file %v: %v", fileName, err.Error())
						}
					}
				}
			}
		}

	}
}
