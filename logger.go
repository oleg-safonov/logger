// Package logger is an example of using the package github.com/oleg-safonov/logwriter .
// Using the logwriter package allows the logger package to be non-blocking and do not slow down the application even with a slow disk.
// The file name is the only necessary parameter for the logger.
// The principle of operation is as follows: upon initialization, logger opens an existing file or creates a new one if necessary.
// On Linux, you can freely move a file while logger writes to it. For example 'mv today.log yesterday.log'
// Next, you need to send a SIGHUP signal to the process, then logger will reopen the today.log file and continue to write to it.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/oleg-safonov/logwriter"
)

const (
	FatalLevel = iota
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

const (
	defaulMaxRecordSize    = 4060
	defaulBuffers          = 1000
	defaulBufferSize       = 4096
	defaulTimeUpdatePeriod = 10 * time.Millisecond
)

var levelStrings map[byte][]byte

func init() {
	levelStrings = make(map[byte][]byte)
	levelStrings[FatalLevel] = []byte("FATAL: ")
	levelStrings[ErrorLevel] = []byte("ERROR: ")
	levelStrings[WarnLevel] = []byte("WARNING: ")
	levelStrings[InfoLevel] = []byte("INFO: ")
	levelStrings[DebugLevel] = []byte("DEBUG: ")
	levelStrings[TraceLevel] = []byte("TRACE: ")
}

type buffer struct {
	buf [defaulBufferSize]byte
	pos int
}

func (b *buffer) clear() {
	b.pos = 0
}

func (b *buffer) get() []byte {
	return b.buf[:b.pos]
}

func (b *buffer) append(data []byte) {
	l := len(data)
	if (b.pos + l) > defaulBufferSize {
		l = defaulBufferSize - b.pos
	}
	if l > 0 {
		copy(b.buf[b.pos:b.pos+l], data)
		b.pos += l
	}
}

// LoggerConfig encapsulates initializing parameters for the Logger.
// Filename is the path to the file that the logger opens for writing.
// Callback WriteErrorHandler is called if an error occurred while writing to the Out.
// Callback SkipHandler is called if there is not enough space in the internal buffer for a new record.
type LoggerConfig struct {
	Filename          string
	SkipHandler       func(int)
	WriteErrorHandler func(io.Writer)
}

// Logger represents an active logging object that generates lines of output to a logwriter.
// Multiple goroutines may invoke methods on a Logger simultaneously.
type Logger struct {
	writer   *logwriter.LogWriter
	buffers  [defaulBuffers]buffer
	bufStack chan int

	skipHandler       func(int)
	writeErrorHandler func(io.Writer)

	muUpdate     sync.Mutex
	currentLevel byte
	timeFormat   string
	timeStr      *[]byte

	filename string
	muReopen sync.Mutex
	file     *os.File

	signalChan chan os.Signal
}

// New creates a new Logger with parameters from LoggerConfig.
func New(config LoggerConfig) (*Logger, error) {
	l := new(Logger)
	l.filename = config.Filename
	l.skipHandler = config.SkipHandler
	l.writeErrorHandler = config.WriteErrorHandler
	f, err := os.OpenFile(l.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return nil, err
	}

	l.currentLevel = InfoLevel
	l.timeFormat = "2006.01.02 15:04:05.00 "
	s := time.Now().Format(l.timeFormat)
	sl := []byte(s)
	l.timeStr = &sl
	l.file = f
	l.writer = logwriter.New(logwriter.LogConfig{Out: l.file,
		SkipHandler:       l.skipHandler,
		WriteErrorHandler: l.writeErrorHandler})

	l.bufStack = make(chan int, defaulBuffers)
	for i := 0; i < defaulBuffers; i++ {
		l.bufStack <- i
	}
	log.SetOutput(l.writer)

	l.signalChan = make(chan os.Signal, 1)
	signal.Notify(l.signalChan, syscall.SIGHUP)
	go l.updateTime()
	go l.signalLoop()
	return l, nil
}

// Set the level of logging. It is possible to set FatalLevel, ErrorLevel, WarnLevel, InfoLevel, DebugLevel, TraceLevel.
// Records with a level below the set level will be ignored when writing. The default is InfoLevel.
func (l *Logger) SetLevel(level byte) error {
	if _, ok := levelStrings[level]; ok != true {
		return fmt.Errorf("Unknown log level %d", level)
	}

	l.muUpdate.Lock()
	l.currentLevel = level
	l.muUpdate.Unlock()
	return nil
}

// SetTimeFormat customizes the time format.
func (l *Logger) SetTimeFormat(layout string) {
	l.muUpdate.Lock()
	l.timeFormat = layout
	l.muUpdate.Unlock()
}

// Reopen waits when all previous records are added to the log and again opens a file with the name LoggerConfig.filename.
func (l *Logger) Reopen() error {
	f, err := os.OpenFile(l.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	l.muReopen.Lock()
	l.writer.Reset(f)
	l.file.Close()
	l.file = f
	l.muReopen.Unlock()
	return nil
}

func (l *Logger) signalLoop() {
	for {
		s := <-l.signalChan
		switch s {
		// kill -SIGHUP XXXX
		case syscall.SIGHUP:
			l.Reopen()
		}
	}
}

func (l *Logger) updateTime() {
	ticker := time.NewTicker(defaulTimeUpdatePeriod)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			l.muUpdate.Lock()
			s := t.Format(l.timeFormat)
			sl := []byte(s)
			l.timeStr = &sl
			l.muUpdate.Unlock()
		}
	}
}

func (l *Logger) getTime() *[]byte {
	l.muUpdate.Lock()
	defer l.muUpdate.Unlock()
	return l.timeStr
}

func (l *Logger) output(level byte, data []byte) {
	l.muUpdate.Lock()
	if level > l.currentLevel {
		l.muUpdate.Unlock()
		return
	}
	l.muUpdate.Unlock()

	i := <-l.bufStack
	defer func(i int) { l.bufStack <- i }(i)
	l.buffers[i].clear()
	l.buffers[i].append(*(l.getTime()))
	l.buffers[i].append(levelStrings[level])
	l.buffers[i].append(data)

	if l.buffers[i].get()[len(l.buffers[i].get())-1] != '\n' {
		l.buffers[i].append([]byte("\n"))
	}

	l.writer.Write(l.buffers[i].get())
}

// LogFatal appends the Fatal prefix to the data string and writes to a file with the Fatal level.
func (l *Logger) LogFatal(data []byte) {
	l.output(FatalLevel, data)
}

// LogError appends the Error prefix to the data string and writes to a file with the Error level.
func (l *Logger) LogError(data []byte) {
	l.output(ErrorLevel, data)
}

// LogWarn appends the Warning prefix to the data string and writes to a file with the Warn level.
func (l *Logger) LogWarn(data []byte) {
	l.output(WarnLevel, data)
}

// LogInfo appends the Info prefix to the data string and writes to a file with the Info level.
func (l *Logger) LogInfo(data []byte) {
	l.output(InfoLevel, data)
}

// LogDebug appends the Debug prefix to the data string and writes to a file with the Debug level.
func (l *Logger) LogDebug(data []byte) {
	l.output(DebugLevel, data)
}

// LogTrace appends the Trace prefix to the data string and writes to a file with the Trace level.
func (l *Logger) LogTrace(data []byte) {
	l.output(TraceLevel, data)
}
