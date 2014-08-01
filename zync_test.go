package main

import "fmt"
import "io"
import "io/ioutil"
import "os"
import "os/exec"
import "path/filepath"
import "strings"
import "testing"
import "time"

var zyncDir, _ = os.Getwd()

// Wraps an io.Writer, adding the specified prefix to each line.
// Used in tests to differentiate server and client output.
type prefixWriter struct {
	out io.Writer
	prefix string
}

func (pw prefixWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range(lines) {
		if line == "" {
			continue
		}
		fmt.Fprintln(pw.out, pw.prefix, line)
	}
	return len(p), nil
}

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
		f, err = os.Create(filepath.Join(dir, name))
	}

	if err != nil {
		panic(err)
	}

	defer f.Close()

	fmt.Fprint(f, content)

	return filepath.Base(f.Name())
}

// Executes zync with the specified arguments in a new temporary directory.
// Returns the temp folder and a channel that can be closed to kill the process
// and clean up the temp folder.
func zyncExecAsync(args ...string) (dir string, sig chan bool) {
	dir = createTempDir()

	zync := filepath.Join(zyncDir, "zync")
	cmd := exec.Command(zync, args...)
	cmd.Dir = dir
	cmd.Stdout = prefixWriter { os.Stdout, "SERVER (OUT)" }
	cmd.Stderr = prefixWriter { os.Stderr, "SERVER (ERR)" }

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

		os.RemoveAll(dir)
	}()

	return
}

// Executes zync with the specified arguments in the specified directory.
func zyncExec(dir string, args ...string) {
	zync := filepath.Join(zyncDir, "zync")
	cmd := exec.Command(zync, args...)
	cmd.Dir = dir
	cmd.Stdout = prefixWriter { os.Stdout, "CLIENT (OUT)" }
	cmd.Stderr = prefixWriter { os.Stderr, "CLIENT (ERR)" }

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

// Creates a temp directory, yields it to the passed function, and then cleans
// it up.
func withTempDir(do func(dir string)) {
	dir := createTempDir()

	defer func () {
		os.RemoveAll(dir)
	}()

	do(dir)
}

// Creates a test folder in the specified directory.
func createDir(root, name string) string {
	name = filepath.Join(root, name)

	err := os.MkdirAll(name, os.ModeDir | 0700)
	if err != nil {
		panic(err)
	}

	return name
}

// Checks that specified file exists in the specified folder and has the
// specified content.
func expectContent(t *testing.T, dir, fname, content string) {
	path := filepath.Join(dir, fname)

	f, err := os.Open(path)
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
		return
	}

	if string(data) != content {
		t.Errorf("Expected %s, read %s.", content, string(data))
	}
}

// Checks that the specified file exists in the specified folder.
func expectExists(t *testing.T, dir, fname string) {
	_, err := os.Stat(filepath.Join(dir, fname))
	if err != nil {
		t.Error(err)
	}
}

// Checks that the specified file exists in the specified folder.
func expectNotExists(t *testing.T, dir, fname string) {
	_, err := os.Stat(filepath.Join(dir, fname))
	if err == nil {
		t.Errorf("Did not expect %s%s to exist.", dir, fname)
	}
}

// The client should send any files the server is missing to it.
func TestSendingFileToServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestSendingFileToServer")
		zyncExec(dir, "-c", "localhost")

		expectContent(t, dir, fname, "TestSendingFileToServer")
		expectContent(t, svrDir, fname, "TestSendingFileToServer")
	})
}

// The client should request any files it is missing from the server.
func TestReceivingFileFromServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	fname := createTestFile(svrDir, "", "TestReceivingFileFromServer")

	withTempDir(func(dir string) {
		zyncExec(dir, "-c", "localhost")
		expectContent(t, svrDir, fname, "TestReceivingFileFromServer")
		expectContent(t, dir, fname, "TestReceivingFileFromServer")
	})
}

// By default, the newer file is kept.
func TestSendingNewerFileToServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestSendingNewerFileToServer1")
		createTestFile(svrDir, fname, "TestSendingNewerFileToServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(filepath.Join(dir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v")
		expectContent(t, dir, fname, "TestSendingNewerFileToServer1")
		expectContent(t, svrDir, fname, "TestSendingNewerFileToServer1")
	})
}

// By default, the newer file is kept.
func TestReceivingNewerFileFromServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestReceivingNewerFileFromServer1")
		createTestFile(svrDir, fname, "TestReceivingNewerFileFromServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(filepath.Join(svrDir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v")
		expectContent(t, dir, fname, "TestReceivingNewerFileFromServer2")
		expectContent(t, svrDir, fname, "TestReceivingNewerFileFromServer2")
	})
}

// If "--keep mine" is specified, the client's file should be used even when
// it is older.
func TestSendingOlderFileToServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestSendingOlderFileToServer1")
		createTestFile(svrDir, fname, "TestSendingOlderFileToServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(filepath.Join(svrDir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v", "-k", "mine")
		expectContent(t, dir, fname, "TestSendingOlderFileToServer1")
		expectContent(t, svrDir, fname, "TestSendingOlderFileToServer1")
	})
}

// If "--keep theirs" is specified, the server's file should be used even when
// it is older.
func TestReceivingOlderFileFromServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestReceivingOlderFileFromServer1")
		createTestFile(svrDir, fname, "TestReceivingOlderFileFromServer2")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(filepath.Join(dir, fname), future, future)

		zyncExec(dir, "-c", "localhost", "-v", "-k", "theirs")
		expectContent(t, dir, fname, "TestReceivingOlderFileFromServer2")
		expectContent(t, svrDir, fname, "TestReceivingOlderFileFromServer2")
	})
}

// If "--keep mine" and "--delete" are specified, files that the client does
// not have will be deleted from the server.
func TestDeletingFileFromServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(svrDir, "", "TestDeletingFileFromServer")
		expectExists(t, svrDir, fname)

		zyncExec(dir, "-c", "localhost", "-v", "-k", "mine", "-d")
		expectNotExists(t, svrDir, fname)
	})
}

// If "--keep theirs" and "--delete" are specified, files that the server does
// not have will be deleted from the server.
func TestDeletingFileFromClient(t *testing.T) {
	_, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		fname := createTestFile(dir, "", "TestDeletingFileFromClient")
		expectExists(t, dir, fname)

		zyncExec(dir, "-c", "localhost", "-v", "-k", "theirs", "-d")
		expectNotExists(t, dir, fname)
	})
}

// If "--keep mine" and "--delete" are specified, folders that the client does
// not have will be deleted from the server along with all of their contents.
func TestDeletingFolderFromServer(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		// .
		// ├── TestFolder1
		// │   ├── TestFile1
		// │   ├── TestFile2
		// │   ├── TestFolder2
		// │   │   ├── TestFile3
		// │   │   └── TestFile4
		// │   └── TestFolder3
		// │       └── TestFile5
		// └── TestFile6

		testFolder1 := createDir(svrDir, "TestFolder1")
		createTestFile(testFolder1, "TestFile1", "TestFile1")
		createTestFile(testFolder1, "TestFile2", "TestFile2")
		testFolder2 := createDir(svrDir, "TestFolder1/TestFolder2")
		createTestFile(testFolder2, "TestFile3", "TestFile3")
		createTestFile(testFolder2, "TestFile4", "TestFile4")
		testFolder3 := createDir(svrDir, "TestFolder1/TestFolder3")
		createTestFile(testFolder3, "TestFile5", "TestFile5")

		createTestFile(svrDir, "TestFile6", "TestFile6")
		createTestFile(dir, "TestFile6", "TestFile6")


		expectExists(t, svrDir, "TestFolder1")
		expectExists(t, svrDir, "TestFolder1/TestFile1")
		expectExists(t, svrDir, "TestFolder1/TestFile2")
		expectExists(t, svrDir, "TestFolder1/TestFolder2")
		expectExists(t, svrDir, "TestFolder1/TestFolder2/TestFile3")
		expectExists(t, svrDir, "TestFolder1/TestFolder2/TestFile4")
		expectExists(t, svrDir, "TestFolder1/TestFolder3")
		expectExists(t, svrDir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, svrDir, "TestFile6")

		expectNotExists(t, dir, "TestFolder1")
		expectNotExists(t, dir, "TestFolder1/TestFile1")
		expectNotExists(t, dir, "TestFolder1/TestFile2")
		expectNotExists(t, dir, "TestFolder1/TestFolder2")
		expectNotExists(t, dir, "TestFolder1/TestFolder2/TestFile3")
		expectNotExists(t, dir, "TestFolder1/TestFolder2/TestFile4")
		expectNotExists(t, dir, "TestFolder1/TestFolder3")
		expectNotExists(t, dir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, dir, "TestFile6")


		zyncExec(dir, "-c", "localhost", "-v", "-k", "mine", "-d")


		expectNotExists(t, svrDir, "TestFolder1")
		expectNotExists(t, svrDir, "TestFolder1/TestFile1")
		expectNotExists(t, svrDir, "TestFolder1/TestFile2")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2/TestFile3")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2/TestFile4")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder3")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, svrDir, "TestFile6")

		expectNotExists(t, dir, "TestFolder1")
		expectNotExists(t, dir, "TestFolder1/TestFile1")
		expectNotExists(t, dir, "TestFolder1/TestFile2")
		expectNotExists(t, dir, "TestFolder1/TestFolder2")
		expectNotExists(t, dir, "TestFolder1/TestFolder2/TestFile3")
		expectNotExists(t, dir, "TestFolder1/TestFolder2/TestFile4")
		expectNotExists(t, dir, "TestFolder1/TestFolder3")
		expectNotExists(t, dir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, dir, "TestFile6")
	})
}

// If "--keep theirs" and "--delete" are specified, folders that the server
// does not have will be deleted from the client along with all of their
// contents.
func TestDeletingFolderFromClient(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		// .
		// ├── TestFolder1
		// │   ├── TestFile1
		// │   ├── TestFile2
		// │   ├── TestFolder2
		// │   │   ├── TestFile3
		// │   │   └── TestFile4
		// │   └── TestFolder3
		// │       └── TestFile5
		// └── TestFile6

		testFolder1 := createDir(dir, "TestFolder1")
		createTestFile(testFolder1, "TestFile1", "TestFile1")
		createTestFile(testFolder1, "TestFile2", "TestFile2")
		testFolder2 := createDir(dir, "TestFolder1/TestFolder2")
		createTestFile(testFolder2, "TestFile3", "TestFile3")
		createTestFile(testFolder2, "TestFile4", "TestFile4")
		testFolder3 := createDir(dir, "TestFolder1/TestFolder3")
		createTestFile(testFolder3, "TestFile5", "TestFile5")

		createTestFile(svrDir, "TestFile6", "TestFile6")
		createTestFile(dir, "TestFile6", "TestFile6")


		expectExists(t, dir, "TestFolder1")
		expectExists(t, dir, "TestFolder1/TestFile1")
		expectExists(t, dir, "TestFolder1/TestFile2")
		expectExists(t, dir, "TestFolder1/TestFolder2")
		expectExists(t, dir, "TestFolder1/TestFolder2/TestFile3")
		expectExists(t, dir, "TestFolder1/TestFolder2/TestFile4")
		expectExists(t, dir, "TestFolder1/TestFolder3")
		expectExists(t, dir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, dir, "TestFile6")

		expectNotExists(t, svrDir, "TestFolder1")
		expectNotExists(t, svrDir, "TestFolder1/TestFile1")
		expectNotExists(t, svrDir, "TestFolder1/TestFile2")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2/TestFile3")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2/TestFile4")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder3")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, svrDir, "TestFile6")


		zyncExec(dir, "-c", "localhost", "-v", "-k", "theirs", "-d")


		expectNotExists(t, svrDir, "TestFolder1")
		expectNotExists(t, svrDir, "TestFolder1/TestFile1")
		expectNotExists(t, svrDir, "TestFolder1/TestFile2")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2/TestFile3")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder2/TestFile4")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder3")
		expectNotExists(t, svrDir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, svrDir, "TestFile6")

		expectNotExists(t, dir, "TestFolder1")
		expectNotExists(t, dir, "TestFolder1/TestFile1")
		expectNotExists(t, dir, "TestFolder1/TestFile2")
		expectNotExists(t, dir, "TestFolder1/TestFolder2")
		expectNotExists(t, dir, "TestFolder1/TestFolder2/TestFile3")
		expectNotExists(t, dir, "TestFolder1/TestFolder2/TestFile4")
		expectNotExists(t, dir, "TestFolder1/TestFolder3")
		expectNotExists(t, dir, "TestFolder1/TestFolder3/TestFile5")
		expectExists(t, dir, "TestFile6")
	})
}

// If the server is run with the "--restrict (-r)" option, it will refuse to
// delete any files, but will accept newer versions of files.
func TestServerRestrictingDelete(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-r", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		createTestFile(svrDir, "TestFile1", "TestFile1")
		createTestFile(svrDir, "TestFile2", "TestFile2a")
		createTestFile(dir, "TestFile2", "TestFile2b")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(filepath.Join(dir, "TestFile2"), future, future)


		expectContent(t, svrDir, "TestFile1", "TestFile1")
		expectContent(t, svrDir, "TestFile2", "TestFile2a")
		expectContent(t, dir, "TestFile2", "TestFile2b")


		zyncExec(dir, "-c", "localhost", "-v", "-k", "mine", "-d")


		expectContent(t, svrDir, "TestFile1", "TestFile1")
		expectContent(t, svrDir, "TestFile2", "TestFile2b")
		expectContent(t, dir, "TestFile2", "TestFile2b")
	})
}

// If the server is run with the "--Restrict (-R)" option, it will refuse to
// delete or overwrite any files.
func TestServerRestrictingAll(t *testing.T) {
	svrDir, svr := zyncExecAsync("-s", "-R", "-v")
	defer close(svr)

	withTempDir(func(dir string) {
		createTestFile(svrDir, "TestFile1", "TestFile1")
		createTestFile(svrDir, "TestFile2", "TestFile2a")
		createTestFile(dir, "TestFile2", "TestFile2b")

		future := time.Now().Add(5 * time.Minute)
		os.Chtimes(filepath.Join(dir, "TestFile2"), future, future)


		expectContent(t, svrDir, "TestFile1", "TestFile1")
		expectContent(t, svrDir, "TestFile2", "TestFile2a")
		expectContent(t, dir, "TestFile2", "TestFile2b")


		zyncExec(dir, "-c", "localhost", "-v", "-k", "mine", "-d")


		expectContent(t, svrDir, "TestFile1", "TestFile1")
		expectContent(t, svrDir, "TestFile2", "TestFile2a")
		expectContent(t, dir, "TestFile2", "TestFile2b")
	})
}
