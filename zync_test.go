package main

import "fmt"
import "io/ioutil"
import "os"
import "os/exec"
import "path"
import "testing"
import "time"

var zyncDir, _ = os.Getwd()

func createTempDir() string {
	name, err := ioutil.TempDir("", "zync")
	if err != nil {
		panic(err)
	}

	return name
}

// Creates a test file in the specified directory. If name is the empty string,
// generates a unique temp file name.
func createTestFile(dir string, name string, content string) (fname string) {
	var f *os.File
	var err error

	if name == "" {
		f, err = ioutil.TempFile(dir, "zync")
	} else {
		f, err = os.Create(path.Join(dir, name))
	}

	if err != nil {
		panic(err)
	}

	defer f.Close()

	fmt.Fprint(f, content)

	return path.Base(f.Name())
}

// Executes zync with the specified arguments in a new temporary directory.
// Returns the temp folder and a channel that can be closed to kill the process
// and clean up the temp folder.
func zyncExecAsync(args ...string) (dir string, sig chan bool) {
	dir = createTempDir()

	zync := path.Join(zyncDir, "zync")
	cmd := exec.Command(zync, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	sig = make(chan bool)
	go func() {
		for _ = range(sig) {}

		err = cmd.Process.Kill()
		if err != nil {
			panic(err)
		}

		err := os.RemoveAll(dir)
		if err != nil {
			panic(err)
		}
	}()

	return
}

// Executes zync with the specified arguments in the specified directory.
func zyncExec(dir string, args ...string) {
	zync := path.Join(zyncDir, "zync")
	cmd := exec.Command(zync, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

// Creates a temp directory, yields it to the passed function, and then cleans
// it up.
func withTempDir(do func(dir string)) {
	dir := createTempDir()

	defer os.RemoveAll(dir)

	do(dir)
}

// Checks that specified file exists in the specified folder with the specified
// content.
func expectContent(t *testing.T, dir, fname, content string) {
	fmt.Println("Looking for", fname, "in", dir)
	f, err := os.Open(path.Join(dir, fname))
	if err != nil {
		t.Error(err)
		return
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
		return
	}

	if string(data) != content {
		t.Errorf("Expected %s, read %s.", content, string(data))
	}
}

// The client should send any files the server is missing to it.
func TestSendingFileToServer(t *testing.T) {
	svrFolder, svr := zyncExecAsync("-s")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestSendingFileToServer")
		zyncExec(dir, "-c", "localhost")

		expectContent(t, svrFolder, fname, "TestSendingFileToServer")
	})
}

// The client should request any files it is missing from the server.
func TestReceivingFileFromServer(t *testing.T) {
	svrFolder, svr := zyncExecAsync("-s")
	defer close(svr)

	fname := createTestFile(svrFolder, "", "TestReceivingFileFromServer")

	withTempDir(func(dir string) {
		zyncExec(dir, "-c", "localhost")
		expectContent(t, dir, fname, "TestReceivingFileFromServer")
	})
}

// By default, the newer file is kept.
func TestSendingNewerFileToServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestSendingNewerFileToServer1")
		createTestFile(svrDir, fname, "TestSendingNewerFileToServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(path.Join(dir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v")
		expectContent(t, svrDir, fname, "TestSendingNewerFileToServer1")
		expectContent(t, dir, fname, "TestSendingNewerFileToServer1")
	})
}

// By default, the newer file is kept.
func TestReceivingNewerFileFromServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestReceivingNewerFileFromServer1")
		createTestFile(svrDir, fname, "TestReceivingNewerFileFromServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(path.Join(svrDir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v")
		expectContent(t, svrDir, fname, "TestReceivingNewerFileFromServer2")
		expectContent(t, dir, fname, "TestReceivingNewerFileFromServer2")
	})
}

// If "--keep mine" is specified, the client's file should be used even when
// it is older.
func TestSendingOlderFileToServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestSendingOlderFileToServer1")
		createTestFile(svrDir, fname, "TestSendingOlderFileToServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(path.Join(svrDir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v", "-k", "mine")
		expectContent(t, svrDir, fname, "TestSendingOlderFileToServer1")
		expectContent(t, dir, fname, "TestSendingOlderFileToServer1")
	})
}

// If "--keep theirs" is specified, the server's file should be used even when
// it is older.
func TestReceivingOlderFileFromServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestReceivingOlderFileFromServer1")
		createTestFile(svrDir, fname, "TestReceivingOlderFileFromServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(path.Join(dir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v", "-k", "theirs")
		expectContent(t, svrDir, fname, "TestReceivingOlderFileFromServer2")
		expectContent(t, dir, fname, "TestReceivingOlderFileFromServer2")
	})
}
