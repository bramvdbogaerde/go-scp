package scp

import (
	"fmt"
	"io"
	"os"
	"path"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	Host         string
	ClientConfig *ssh.ClientConfig
	Session      *ssh.Session
}

// Connects to the remote SSH server, returns error if it couldn't establish a session to the SSH server
func (a *Client) Connect() error {
	client, err := ssh.Dial("tcp", a.Host, a.ClientConfig)
	if err != nil {
		return err
	}

	a.Session, err = client.NewSession()
	if err != nil {
		return err
	}
	return nil
}

// Copies the contents of an os.File to a remote location
func (a *Client) CopyFile(file *os.File, remotePath string, permissions string) {
	filename := path.Base(remotePath)
	directory := path.Dir(remotePath)
	stat, _ := file.Stat()

	go func() {
		w, _ := a.Session.StdinPipe()
		defer w.Close()
		fmt.Fprintln(w, "C"+permissions, stat.Size(), filename)
		io.Copy(w, file)
		fmt.Fprintln(w, "\x00")
	}()

	a.Session.Run("/usr/bin/scp -t " + directory)
}
