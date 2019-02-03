/* Copyright (c) 2018 Bram Vandenbogaerde
 * You may use, distribute or modify this code under the
 * terms of the Mozilla Public License 2.0, which is distributed
 * along with the source code.
 */
package scp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	Host         string
	ClientConfig *ssh.ClientConfig
	Session      *ssh.Session
	Conn         ssh.Conn
	Timeout      time.Duration
}

// Connects to the remote SSH server, returns error if it couldn't establish a session to the SSH server
func (a *Client) Connect() error {
	client, err := ssh.Dial("tcp", a.Host, a.ClientConfig)
	if err != nil {
		return err
	}

	a.Conn = client.Conn
	a.Session, err = client.NewSession()
	if err != nil {
		return err
	}
	return nil
}

//Copies the contents of an os.File to a remote location, it will get the length of the file by looking it up from the filesystem
func (a *Client) CopyFromFile(file os.File, remotePath string, permissions string) error {
	stat, _ := file.Stat()
	return a.Copy(&file, remotePath, permissions, stat.Size())
}

// Copies the contents of an io.Reader to a remote location, the length is determined by reading the io.Reader until EOF
// if the file length in know in advance please use "Copy" instead
func (a *Client) CopyFile(fileReader io.Reader, remotePath string, permissions string) error {
	contents_bytes, _ := ioutil.ReadAll(fileReader)
	bytes_reader := bytes.NewReader(contents_bytes)

	return a.Copy(bytes_reader, remotePath, permissions, int64(len(contents_bytes)))
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	for {
		select {
		case <-c:
			return false // completed normally
		case <-time.After(timeout):
			if timeout.Nanoseconds() == 0 {
				continue
			}
			return true // timed out
		}
	}

}

// CopyFromRemote copies the contents of a remote file into an io.Reader
func (a *Client) CopyFromRemote(remotePath string) (io.Reader, os.FileMode, error) {
	errCh := make(chan error, 2)

	c := &Command{}
	readerCh := make(chan io.Reader, 1)

	r, _, w, err := pipes(a.Session)
	if err != nil {
		return nil, 0, err
	}

	prepReader := func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer w.Close()

		err = readCommand(r, w, c)
		if err != nil && err != io.EOF {
			errCh <- err
			return
		}

		// prep to recieve
		_, err = w.Write([]byte{0})
		if err != nil {
			errCh <- err
			return
		}

		// redirect our reader
		readerCh <- r
	}

	syncFile := func(wg *sync.WaitGroup) {
		defer wg.Done()

		// start the transfer (and ignore error codes)
		a.Session.Run("/usr/bin/scp -qf " + remotePath)
	}

	wg := a.scpFunc(prepReader, syncFile, errCh, 2)

	if waitTimeout(wg, a.Timeout) {
		return nil, 0, errors.New("timeout when downloading files")
	}

	close(errCh)
	for err := range errCh {
		if err != nil {
			return nil, 0, err
		}
	}

	return <-readerCh, c.Permissions, nil
}

// Copies the contents of an io.Reader to a remote location
func (a *Client) Copy(r io.Reader, remotePath string, permissions string, size int64) error {
	filename := path.Base(remotePath)
	directory := path.Dir(remotePath)
	errCh := make(chan error, 2)

	syncFile := func(wg *sync.WaitGroup) {
		defer wg.Done()

		err := a.Session.Run("/usr/bin/scp -qt " + directory)
		if err != nil {
			errCh <- err
			return
		}
	}

	copyFile := func(wg *sync.WaitGroup) {
		defer wg.Done()
		w, err := a.Session.StdinPipe()
		if err != nil {
			errCh <- err
			return
		}
		defer w.Close()

		_, err = fmt.Fprintln(w, "C"+permissions, size, filename)
		if err != nil {
			errCh <- err
			return
		}

		_, err = io.Copy(w, r)
		if err != nil {
			errCh <- err
			return
		}

		_, err = fmt.Fprint(w, "\x00")
		if err != nil {
			errCh <- err
			return
		}
	}

	wg := a.scpFunc(copyFile, syncFile, errCh, 2)

	if waitTimeout(wg, a.Timeout) {
		return errors.New("timeout when upload files")
	}
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Client) Close() {
	a.Session.Close()
	a.Conn.Close()
}

func (a *Client) scpFunc(fileFunc func(*sync.WaitGroup), syncFunc func(*sync.WaitGroup), errCh chan error, wgSize int) *sync.WaitGroup {
	wg := sync.WaitGroup{}
	wg.Add(wgSize)

	go func() {
		fileFunc(&wg)
	}()

	go func() {
		syncFunc(&wg)
	}()

	return &wg
}

// readCommand readies a buffer and initiates listening for an SCP command, and returns
// the Command when parsed. w is not closed by readCommand
func readCommand(r io.Reader, w io.WriteCloser, c *Command) error {
	// recieve remote command input
	buf := bytes.Buffer{}

	// write a null byte to signal we are ready to receive data
	_, err := w.Write([]byte{0})
	if err != nil && err != io.EOF {
		return err
	}

	for {
		cmd := make([]byte, 64)
		_, err := r.Read(cmd)
		if err != nil && err != io.EOF {
			return err
		} else if err == io.EOF {
			break
		}

		if string(cmd) == "" {
			continue
		}
		_, err = buf.Write(cmd)
		if err != nil {
			return err
		}

		err = c.UnmarshalText(buf.Bytes())
		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("no scp cmd found")
}

func pipes(s *ssh.Session) (io.Reader, io.Reader, io.WriteCloser, error) {
	w, err := s.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	r, err := s.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	e, err := s.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	return r, e, w, err
}
