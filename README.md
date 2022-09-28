# Dirkeeper - A tool to watch and cleanup directories

Dirkeeper is a tool to manage local directories with some useful commands.
At the moment the following commands are available:
- cleanold: cleans files older than a specified number of days
- match: matches files inside a folder and runs actions on them
- watch: watch one or more directories for the creation of new files and executes an action if the file name matches a condition

## Command syntax
```
Directory management utilities

Usage:
  dirkeeper [command]

Available Commands:
  cleanold    clean old files
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  match       match and process files
  watch       watch for new files and process them based on config rules

Flags:
  -h, --help   help for dirkeeper

Use "dirkeeper [command] --help" for more information about a command.
```

### cleanold command
```
clean old files

Usage:
  dirkeeper cleanold [flags]

Flags:
  -d, --directory strings   Directory to cleanup
      --dry-run             Only check for old files without deleting
  -h, --help                help for cleanold
      --max-age int         Maximum age of the file in days
```

### match command
```
match and process files

Usage:
  dirkeeper match [flags]

Flags:
  -a, --action string      Action to execute
      --dest-dir string    Destination directory
  -d, --directory string   Base directory
      --dry-run            Do not execute action
  -h, --help               help for match
      --max-age int        Max file age in minutes
      --pattern strings    File name pattern
      --prefix strings     File name prefix
      --suffix strings     File name suffix
```

### watch command
Inside the `config` folder you can find an example configuration file
```
watch for new files and process them based on config rules

Usage:
  dirkeeper watch [flags]

Flags:
  -c, --config string   Config file
      --debug           Enable debug log
      --frequency int   Watch frequency (default 10s) (default 10)
  -h, --help            help for watch
```