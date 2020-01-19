Copy files over SCP with Go
=============================
[![Go Report Card](https://goreportcard.com/badge/bramvdbogaerde/go-scp)](https://goreportcard.com/report/bramvdbogaerde/go-scp) [![](https://godoc.org/github.com/bramvdbogaerde/go-scp?status.svg)](https://godoc.org/github.com/bramvdbogaerde/go-scp)

This package makes it very easy to copy files over scp in Go.
It uses the golang.org/x/crypto/ssh package to establish a secure connection to a remote server in order to copy the files via the SCP protocol.

### Example usage

```go
package main

import (
	"fmt"
	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
	"os"
)

func main() {
	// Use SSH key authentication from the auth package
	// we ignore the host key in this example, please change this if you use this library
	clientConfig, _ := auth.PrivateKey("username", "/path/to/rsa/key", ssh.InsecureIgnoreHostKey())

	// For other authentication methods see ssh.ClientConfig and ssh.AuthMethod

	// Create a new SCP client
	client := scp.NewClient("example.com:22", &clientConfig)

	// Connect to the remote server
	err := client.Connect()
	if err != nil {
		fmt.Println("Couldn't establish a connection to the remote server ", err)
		return
	}

	// Open a file
	f, _ := os.Open("/path/to/local/file")

	// Close client connection after the file has been copied
	defer client.Close()

	// Close the file after it has been copied
	defer f.Close()

	// Finaly, copy the file over
	// Usage: CopyFile(fileReader, remotePath, permission)

	err = client.CopyFile(f, "/home/server/test.txt", "0655")

	if err != nil {
		fmt.Println("Error while copying file ", err)
	}
}
```

### License

This library is licensed under the Mozilla Public License 2.0.    
A copy of the license is provided in the `LICENSE.txt` file.

Copyright (c) 2020 Bram Vandenbogaerde
