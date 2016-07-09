package scp

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"path"
)

type Client struct {
	Host         string
	ClientConfig *ssh.ClientConfig
	Session      *ssh.Session
}

// Connects to the remote SSH server, returns error if it couldn't establisch a session to the SSH server
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

// Copies the contents of an io.Reader to a remote location
func (a *Client) CopyFile(fileReader io.Reader, remotePath string, permissions string) {
	contents_bytes, _ := ioutil.ReadAll(fileReader)
	contents := string(contents_bytes)
	filename := path.Base(remotePath)
	directory := path.Dir(remotePath)

	go func() {
		w, _ := a.Session.StdinPipe()
		defer w.Close()
		fmt.Fprintln(w, "C"+permissions, len(contents), filename)
		fmt.Fprintln(w, contents)
		fmt.Fprintln(w, "\x00")
	}()

	a.Session.Run("/usr/bin/scp -t " + directory)
}
