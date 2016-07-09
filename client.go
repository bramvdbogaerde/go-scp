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

func (a *Client) CopyFile(fileReader io.Reader, remotePath string, permissions string) {
	contents := string(ioutil.ReadAll(fileReader))
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
