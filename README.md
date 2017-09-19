Copy files over SCP with Go
=============================
[![](https://godoc.org/github.com/bramvdbogaerde/go-scp?status.svg)](https://godoc.org/github.com/bramvdbogaerde/go-scp)

This package makes it very easy to copy files over scp in Go.
It uses the golang.org/x/crypto/ssh package to establish a secure connection to a remote server in order to copy the files via the SCP protocol.

### Example usage

```
package main

import(
	"fmt"
	"os"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/bramvdbogaerde/go-scp"
        "golang.org/x/crypto/ssh"
)

func main(){
	// Use SSH key authentication from the auth package
        // we ignore the host key in this example, please change this if you use this library
	clientConfig, _ := auth.PrivateKey("username", "/path/to/rsa/key", ssh.InsecureIgnoreHostKey())
	
	// For other authentication methods see ssh.ClientConfig and ssh.AuthMethod

	// Create a new SCP client
	client := scp.NewClient("example.com:22", &clientConfig)
	
	// Connect to the remote server
	err := client.Connect()
	if err != nil{
		fmt.Println("Couldn't establisch a connection to the remote server ", err)
		return		
	}

	// Open a file
	f, _ := os.Open("/path/to/local/file")

	// Close session after the file has been copied
	defer client.Session.Close()
	
	// Close the file after it has been copied
	defer f.Close()
	
	// Finaly, copy the file over
	// Usage: CopyFile(fileReader, remotePath, permission)

	client.CopyFile(f, "/path/to/remote/file", "0655")
}
```
