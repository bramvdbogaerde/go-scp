package tests

import(
   "testing"
   "io/ioutil"
   "strings"
   "golang.org/x/crypto/ssh"
   "os"
   scp "github.com/bramvdbogaerde/go-scp"
   "github.com/bramvdbogaerde/go-scp/auth"
)

// This test, tests the basic functionality of the library: copying files
// it assumes that a docker container is running an SSH server at port 
// 2244 using password authentication.
//
// It also assumes that the directory /results is writable within that container
// and is mapped to the tmp/ directory within this directory. 
func TestCopy(t *testing.T) {
   // Use SSH key authentication from the auth package
   // we ignore the host key in this example, please change this if you use this library
   clientConfig, _ := auth.PasswordKey("bram", "test", ssh.InsecureIgnoreHostKey())

   // For other authentication methods see ssh.ClientConfig and ssh.AuthMethod

   // Create a new SCP client
   client := scp.NewClient("127.0.0.1:2244", &clientConfig)

   // Connect to the remote server
   err := client.Connect()
   if err != nil {
            t.Errorf("Couldn't establish a connection to the remote server %s", err)
            return
   }

   // Open a file
   f, _ := os.Open("./input.txt")

   // Close client connection after the file has been copied
   defer client.Close()

   // Close the file after it has been copied
   defer f.Close()

   // Finaly, copy the file over
   // Usage: CopyFile(fileReader, remotePath, permission)

   err = client.CopyFile(f, "/data/output.txt", "0655")

   if err != nil {
            t.Errorf("Error while copying file %s", err)
   }

   content, err := ioutil.ReadFile("./tmp/output.txt")
   if err != nil {
            t.Errorf("Test has failed, file could not be opened")
   }

   text := string(content)
   expected := "It Works"
   if strings.Compare(text, expected) == 0 {
            t.Errorf("Got different text than expected, expected \"%s\" got, \"%s\"", expected, text)
   }
}
