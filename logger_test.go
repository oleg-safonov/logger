package logger

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func ExampleLogger() {
	log, err := New(LoggerConfig{Filename: "./Example.log"})
	if err != nil {
		os.Exit(1)
	}

	log.SetTimeFormat("06.01.02 15'04'05.00")
	log.SetLevel(DebugLevel)

	log.LogDebug([]byte("example string"))
}

func TestCreateLogger(t *testing.T) {
	os.Remove("./TestCreateLogger.log")
	lg, _ := New(LoggerConfig{Filename: "./TestCreateLogger.log"})
	lg.LogTrace([]byte("LogTrace"))
	lg.LogDebug([]byte("LogDebug"))
	lg.LogInfo([]byte("LogInfo"))
	lg.LogWarn([]byte("LogWarn"))
	lg.LogError([]byte("LogError"))
	lg.LogFatal([]byte("LogFatal"))
	time.Sleep(300 * time.Millisecond)
	file, _ := os.Open("./TestCreateLogger.log")
	all, _ := ioutil.ReadAll(file)

	if strings.Contains(string(all), "LogTrace") {
		t.Error("Expected absebce LogTrace, got\n", string(all))
	}

	if strings.Contains(string(all), "LogDebug") {
		t.Error("Expected absebce LogDebug, got\n", string(all))
	}

	if !strings.Contains(string(all), "LogInfo") {
		t.Error("Expected presence LogInfo, got\n", string(all))
	}

	if !strings.Contains(string(all), "LogWarn") {
		t.Error("Expected presence LogWarn, got\n", string(all))
	}

	if !strings.Contains(string(all), "LogError") {
		t.Error("Expected presence LogError, got\n", string(all))
	}

	if !strings.Contains(string(all), "LogFatal") {
		t.Error("Expected presence LogFatal, got\n", string(all))
	}
	os.Remove("./TestCreateLogger.log")
}

func TestSetLevel(t *testing.T) {
	os.Remove("./TestSetLevel.log")
	lg, _ := New(LoggerConfig{Filename: "./TestSetLevel.log"})
	lg.LogTrace([]byte("LogTrace"))
	lg.LogDebug([]byte("LogDebug"))
	lg.SetLevel(DebugLevel)
	lg.LogTrace([]byte("LogTrace"))
	lg.LogDebug([]byte("LogDebug"))
	time.Sleep(300 * time.Millisecond)
	file, _ := os.Open("./TestSetLevel.log")
	all, _ := ioutil.ReadAll(file)

	if strings.Contains(string(all), "LogTrace") {
		t.Error("Expected absebce LogTrace, got\n", string(all))
	}

	if strings.Count(string(all), "LogDebug") != 1 {
		t.Error("Expected once LogDebug, got\n", string(all))
	}

	err := lg.SetLevel(255)

	if err == nil {
		t.Error("Expected Unknown log level 255, got", err)
	}
	os.Remove("./TestSetLevel.log")
}

func TestFailCreateLogger(t *testing.T) {
	_, err := New(LoggerConfig{Filename: "/"})
	if err == nil {
		t.Error("Expected error = open /: is a directory, got", err)
	}

	lg, err := New(LoggerConfig{Filename: "./TestFailCreateLogger.log"})
	lg.filename = "/"
	err = lg.Reopen()

	if err == nil {
		t.Error("Expected error = open /: is a directory, got", err)
	}
	os.Remove("./TestFailCreateLogger.log")
}

func TestReopen(t *testing.T) {
	os.Remove("./TestReopen1.log")
	os.Remove("./TestReopen2.log")
	lg, _ := New(LoggerConfig{Filename: "./TestReopen1.log"})

	lg.LogInfo([]byte("LogInfo1"))
	lg.LogInfo([]byte("LogInfo2"))
	os.Rename("./TestReopen1.log", "./TestReopen2.log")
	lg.Reopen()
	lg.LogInfo([]byte("LogInfo3"))
	lg.LogInfo([]byte("LogInfo4"))
	time.Sleep(300 * time.Millisecond)
	file1, _ := os.Open("./TestReopen1.log")
	all1, _ := ioutil.ReadAll(file1)

	if strings.Contains(string(all1), "LogInfo1") {
		t.Error("Expected absebce LogInfo1, got\n", string(all1))
	}

	if !strings.Contains(string(all1), "LogInfo3") {
		t.Error("Expected presence LogInfo3, got\n", string(all1))
	}

	file2, _ := os.Open("./TestReopen2.log")
	all2, _ := ioutil.ReadAll(file2)

	if strings.Contains(string(all2), "LogInfo4") {
		t.Error("Expected absebce LogInfo4, got\n", string(all2))
	}

	if !strings.Contains(string(all2), "LogInfo2") {
		t.Error("Expected presence LogInfo2, got\n", string(all2))
	}

	os.Remove("./TestReopen1.log")
	os.Remove("./TestReopen2.log")
}

func TestTimeFormat(t *testing.T) {
	os.Remove("./TestTimeFormat.log")

	lg, _ := New(LoggerConfig{Filename: "./TestTimeFormat.log"})
	lg.SetTimeFormat("06.01.02 15'04'05")
	time.Sleep(20 * time.Millisecond)

	s1 := time.Now().Format("06.01.02 15'04'05")
	time.Sleep(defaulTimeUpdatePeriod)
	tm := lg.getTime()
	tm_str := string(*tm)
	s2 := time.Now().Format("06.01.02 15'04'05")

	if !(s1 == tm_str || s2 == tm_str) {
		t.Error("Expected time", s1, "or", s2, " got ", tm_str)
	}

	os.Remove("./TestTimeFormat.log")
}

func TestBuffer(t *testing.T) {
	var buf buffer
	for i := 0; i < defaulBufferSize-1; i++ {
		buf.append([]byte("0"))
	}

	buf.append([]byte("12"))

	if string(buf.get()[defaulBufferSize-5:]) != "00001" {
		t.Error("Expected 00001, got\n", string(buf.get()[defaulBufferSize-5:]))
	}
}

func BenchmarkLogSequential(b *testing.B) {
	os.Remove("./BenchmarkLogSequential.log")
	lg, _ := New(LoggerConfig{Filename: "./BenchmarkLogSequential.log"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.LogInfo([]byte("TEST"))
	}
	os.Remove("./BenchmarkLogSequential.log")
}
