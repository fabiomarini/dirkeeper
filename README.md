# Dirkeeper - A tool to watch and cleanup directories

Dirkeeper is a tool to manage local directories with some useful commands.
At the moment the following commands are available:
- cleanold: cleans files older than a specified number of days
- match: matches files inside a folder and runs actions on them
- watch: watch one or more directories for the creation of new files and executes an action if the file name matches a condition

## Command syntax
```shell
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
```shell
clean old files

Usage:
  dirkeeper cleanold [flags]

Flags:
  -d, --directory strings   List of directories to cleanup
      --dry-run             Only check for old files without deleting
  -h, --help                help for cleanold
      --max-age int         Maximum age of the file in days
```

### match command
```shell
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
      --pattern strings    List of file name patterns
      --prefix strings     List of file name prefixes
      --suffix strings     List of file name suffixes
```

### watch command
Inside the `config` folder you can find an example configuration file
```shell
watch for new files and process them based on config rules

Usage:
  dirkeeper watch [flags]

Flags:
  -c, --config string   Config file
      --debug           Enable debug log
      --frequency int   Watch frequency in seconds (default 10)
  -h, --help            help for watch
```

### freespace command
Checks the available free space on the specified path, and if below a given threshold, sends a notification email.

This command is meant to be run by cron to periodically check the free space.
```shell
check free disk space

Usage:
  dirkeeper freespace [flags]

Flags:
      --email                  Send email notification
      --email-to strings       Email address to send notification
  -h, --help                   help for freespace
      --limit float            Limit percentage (default 15)
  -p, --path string            Path to check (default "/")
      --quiet                  Do not print notification
      --smtp-from string       SMTP from
      --smtp-password string   SMTP password
      --smtp-port int16        SMTP port (default 25)
      --smtp-server string     SMTP server
      --smtp-subject string    SMTP subject
      --smtp-tls               Use TLS
      --smtp-user string       SMTP user
```