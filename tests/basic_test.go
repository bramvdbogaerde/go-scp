package tests

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

func establishConnection(t *testing.T) scp.Client {
	// Use SSH key authentication from the auth package.
	// During testing we ignore the host key, don't to that when you use this.
	clientConfig, _ := auth.PasswordKey("bram", "test", ssh.InsecureIgnoreHostKey())

	// Create a new SCP client.
	client := scp.NewClient("127.0.0.1:2244", &clientConfig)

	// Connect to the remote server.
	err := client.Connect()
	if err != nil {
		t.Fatalf("Couldn't establish a connection to the remote server: %s", err)
	}
	return client
}

// TestCopy tests the basic functionality of copying a file to the remote
// destination.
//
// It assumes that a Docker container is running an SSH server at port 2244
// that is using password authentication. It also assumes that the directory
// /data is writable within that container and is mapped to ./tmp/ within the
// directory the test is run from.
func TestCopy(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/upload_file.txt")
	defer f.Close()

	// Create a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	filename := "Exöt1ç uploaded file.txt"

	// Finaly, copy the file over.
	// Usage: CopyFile(fileReader, remotePath, permission).
	err := client.CopyFile(context.Background(), f, "/data/"+filename, "0777")
	if err != nil {
		t.Errorf("Error while copying file: %s", err)
	}

	// Read what the receiver have written to disk.
	content, err := ioutil.ReadFile("./tmp/" + filename)
	if err != nil {
		t.Errorf("Result file could not be read: %s", err)
	}

	text := string(content)
	expected := "It Works\n"
	if strings.Compare(text, expected) != 0 {
		t.Errorf("Got different text than expected, expected %q got, %q", expected, text)
	}
}

// TestDownloadFile tests the basic functionality of copying a file from the
// remote destination.
//
// It assumes that a Docker container is running an SSH server at port 2244
// that is using password authentication. It also assumes that the directory
// /data is writable within that container and is mapped to ./tmp/ within the
// directory the test is run from.
func TestDownloadFile(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/input.txt")
	defer f.Close()

	// Create a local file to write to.
	f, err := os.OpenFile("./tmp/output.txt", os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		t.Errorf("Couldn't open the output file")
	}
	defer f.Close()

	// Use a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	err = client.CopyFromRemote(context.Background(), f, "/input/Exöt1ç download file.txt.txt")
	if err != nil {
		t.Errorf("Copy failed from remote")
	}

	content, err := ioutil.ReadFile("./tmp/output.txt")
	if err != nil {
		t.Errorf("Result file could not be read: %s", err)
	}

	text := string(content)
	expected := "It works for download!\n"
	if strings.Compare(text, expected) != 0 {
		t.Errorf("Got different text than expected, expected %q got, %q", expected, text)
	}
}

// TestTimeoutDownload tests that a timeout error is produced if the file is not copied in the given
// amount of time.
func TestTimeoutDownload(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()
	client.Timeout = 1 * time.Millisecond

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/upload_file.txt")
	defer f.Close()

	// Create a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	filename := "Exöt1ç uploaded file.txt"

	err := client.CopyFile(context.Background(), f, "/data/"+filename, "0777")
	if err != context.DeadlineExceeded {
		t.Errorf("Expected a timeout error but got succeeded without error")
	}
}

// TestContextCancelDownload tests that a a copy is immediately cancelled if we call context.cancel()
func TestContextCancelDownload(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/upload_file.txt")
	defer f.Close()

	// Create a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	filename := "Exöt1ç uploaded file.txt"

	err := client.CopyFile(ctx, f, "/data/"+filename, "0777")
	if err != context.Canceled {
		t.Errorf("Expected a canceled error but transfer succeeded without error")
	}
}
