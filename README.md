# logger golang
Package logger is an example of using the package [logwriter](https://github.com/oleg-safonov/logwriter).
Using the logwriter package allows the logger package to be non-blocking and do not slow down the application even with a slow disk.
The file name is the only necessary parameter for the logger.
The principle of operation is as follows: upon initialization, logger opens an existing file or creates a new one if necessary.
On Linux, you can freely move a file while logger writes to it. For example 'mv today.log yesterday.log'
Next, you need to send a SIGHUP signal to the process, then logger will reopen the today.log file and continue to write to it.

## Usage
```
	log, err := New(LoggerConfig{Filename: "./Example.log"})
	if err != nil {
		os.Exit(1)
	}

	log.SetTimeFormat("06.01.02 15'04'05.00")
	log.SetLevel(DebugLevel)

	log.LogDebug([]byte("example string"))
```
# Installation
```
go get github.com/oleg-safonov/logger
```
