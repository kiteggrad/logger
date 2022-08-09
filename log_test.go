package logger

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCatchFatal(t *testing.T) {
	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGINT)

	newLogger(t, Config{}).Fatal("fatal")

	// We can get a signal with a little delay
	time.Sleep(10 * time.Millisecond)

	select {
	case s := <-term:
		if s != syscall.SIGINT {
			t.Errorf("want %s signal, got %s", syscall.SIGINT, s)
		}
	default:
		t.Error("didn't get interrupt signal")
	}
}

func TestWriteToFile(t *testing.T) {
	files := createTempFiles(t, "1.log", "2.log")

	t.Logf("use temp files: %v", strings.Join(files, ", "))

	log := newLogger(t, Config{Files: files})

	log.Info(1, 2, 3)

	for _, filename := range files {
		data := readFile(t, filename)
		if len(data) == 0 {
			t.Errorf("no data in file %s", filename)
		}
	}
}

func TestLevelChange(t *testing.T) {
	filename := createTempFiles(t, "1.log")[0]
	log := newLogger(t, Config{Files: []string{filename}})

	var linesCount int

	linesCount += 3 // 3 lines
	log.SetLevel("debug")
	log.Debug(0)
	log.Debug(0)
	log.Info(0)

	// 0 lines
	log.SetLevel("error")
	log.Debug(0)
	log.Info(0)

	linesCount += 2 // 2 lines
	log.SetLevel("info")
	log.Debug(0)
	log.Info(0)
	log.Error(0)

	linesCount += 1 // 1 lines (debug is the minimum level)
	log.SetLevel("trace")
	log.Debug(0)

	data := readFile(t, filename)

	n := bytes.Count(data, []byte("\n"))
	if n != linesCount {
		t.Errorf("want %d messages, got %d", linesCount, n)
	}
}

func TestWithFields(t *testing.T) {
	filename := createTempFiles(t, "1.log")[0]
	log := newLogger(t, Config{Files: []string{filename}})

	expectedMsgs := []string{
		`info	{"error": "some error"}`,
		`debug	{"error": "some error", "key": "value"}`,
		`warn	{"error": "some error"}`,
		`helloworld	{"a": 1, "b": 2, "c": 3}`,
		`hello world	{"a": 1, "b": 2, "c": 3}`,
	}

	l1 := log.WithError(errors.New("some error"))
	l1.Info("info")
	l1.WithField("key", "value").Debug("debug")
	l1.Warn("warn")

	l2 := log.WithField("a", 1).WithField("b", 2).WithFields(map[string]interface{}{"c": 3})
	l2.Error("hello", "world")
	l2.Errorln("hello", "world")

	data := readFile(t, filename)
	scan := bufio.NewScanner(bytes.NewBuffer(data))
	for i := 0; scan.Scan(); i++ {
		want := expectedMsgs[i]
		got := scan.Text()
		if !strings.HasSuffix(got, expectedMsgs[i]) {
			t.Errorf("invalid msg on line #%d: want %s, got %s", i+1, want, got)
		}
	}
}

func TestCaller(t *testing.T) {
	// Check only filepath. Line numbers are too unreliable
	const callerPath = "log_test.go"

	filename := createTempFiles(t, "1.log")[0]
	log := newLogger(t, Config{Files: []string{filename}})

	log.Debug("1")
	log.Debugf("1")
	log.Debugln("1")
	log.Info("1")
	log.Infof("1")
	log.Infoln("1")
	log.Warn("1")
	log.Warnf("1")
	log.Warnln("1")
	log.Warning("1")
	log.Warningf("1")
	log.Warningln("1")
	log.Error("1")
	log.Errorf("1")
	log.Errorln("1")
	// log.Fatal("1")
	// log.Fatalf("1")
	// log.Fatalln("1")
	// log.Panic("1")
	// log.Panicf("1")
	// log.Panicln("1")
	log.Print("1")
	log.Printf("1")
	log.Println("1")

	data := readFile(t, filename)
	scan := bufio.NewScanner(bytes.NewBuffer(data))
	for i := 0; scan.Scan(); i++ {
		line := scan.Text()
		if !strings.Contains(line, callerPath) {
			t.Errorf("line #%d has wrong caller path: %s", i+1, line)
		}
	}
}

func newLogger(t *testing.T, cfg Config) *Logger {
	t.Helper()

	log, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return log
}

func createTempFiles(t *testing.T, filenames ...string) (filepaths []string) {
	t.Helper()

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	t.Logf("use temp dir: %s", dir)

	for _, name := range filenames {
		filepaths = append(filepaths, path.Join(dir, name))
	}
	return filepaths
}

func readFile(t *testing.T, filename string) []byte {
	t.Helper()

	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
