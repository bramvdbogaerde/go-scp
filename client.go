/* Copyright (c) 2021 Bram Vandenbogaerde And Contributors
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

type PassThru func(r io.Reader, total int64) io.Reader

type Client struct {
	// the host to connect to
	Host string

	// the client config to use
	ClientConfig *ssh.ClientConfig

	// stores the SSH session while the connection is running
	Session *ssh.Session

	// stores the SSH connection itself in order to close it after transfer
	Conn ssh.Conn

	// the maximal amount of time to wait for a file transfer to complete
	Timeout time.Duration

	// the absolute path to the remote SCP binary
	RemoteBinary string
}

// Connects to the remote SSH server, returns error if it couldn't establish a session to the SSH server
func (a *Client) Connect() error {
	if a.Session != nil {
		return nil
	}

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

// Copies the contents of an os.File to a remote location, it will get the length of the file by looking it up from the filesystem
func (a *Client) CopyFromFile(file os.File, remotePath string, permissions string) error {
	return a.CopyFromFilePassThru(file, remotePath, permissions, nil)
}

// Copies the contents of an os.File to a remote location, it will get the length of the file by looking it up from the filesystem.
// Access copied bytes by providing a PassThru reader factory
func (a *Client) CopyFromFilePassThru(file os.File, remotePath string, permissions string, passThru PassThru) error {
	stat, _ := file.Stat()
	return a.CopyPassThru(&file, remotePath, permissions, stat.Size(), passThru)
}

// Copies the contents of an io.Reader to a remote location, the length is determined by reading the io.Reader until EOF
// if the file length in know in advance please use "Copy" instead
func (a *Client) CopyFile(fileReader io.Reader, remotePath string, permissions string) error {
	return a.CopyFilePassThru(fileReader, remotePath, permissions, nil)
}

// Copies the contents of an io.Reader to a remote location, the length is determined by reading the io.Reader until EOF
// if the file length in know in advance please use "Copy" instead.
// Access copied bytes by providing a PassThru reader factory
func (a *Client) CopyFilePassThru(fileReader io.Reader, remotePath string, permissions string, passThru PassThru) error {
	contents_bytes, _ := ioutil.ReadAll(fileReader)
	bytes_reader := bytes.NewReader(contents_bytes)

	return a.CopyPassThru(bytes_reader, remotePath, permissions, int64(len(contents_bytes)), passThru)
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	if timeout > 0 {
		select {
		case <-c:
			return false // completed normally
		case <-time.After(timeout):
			return true // timed out
		}
	} else {
		// only wait for waitgroup to complete
		select {
		case <-c:
			return false
		}
	}
}

// Checks the response it reads from the remote, and will return a single error in case
// of failure
func checkResponse(r io.Reader) error {
	response, err := ParseResponse(r)
	if err != nil {
		return err
	}

	if response.IsFailure() {
		return errors.New(response.GetMessage())
	}

	return nil

}

// Copies the contents of an io.Reader to a remote location
func (a *Client) Copy(r io.Reader, remotePath string, permissions string, size int64) error {
	return a.CopyPassThru(r, remotePath, permissions, size, nil)
}

// Copies the contents of an io.Reader to a remote location.
// Access copied bytes by providing a PassThru reader factory
func (a *Client) CopyPassThru(r io.Reader, remotePath string, permissions string, size int64, passThru PassThru) error {
	stdout, err := a.Session.StdoutPipe()
	if err != nil {
		return err
	}

	if passThru != nil {
		r = passThru(r, size)
	}

	filename := path.Base(remotePath)

	wg := sync.WaitGroup{}
	wg.Add(2)

	errCh := make(chan error, 2)

	go func() {
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

		if err = checkResponse(stdout); err != nil {
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

		if err = checkResponse(stdout); err != nil {
			errCh <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		err := a.Session.Run(fmt.Sprintf("%s -qt %q", a.RemoteBinary, remotePath))
		if err != nil {
			errCh <- err
			return
		}
	}()

	if waitTimeout(&wg, a.Timeout) {
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

// Copy a file from the remote to the local file given by the `file`
// parameter. Use `CopyFromRemotePassThru` if a more generic writer
// is desired instead of writing directly to a file on the file system.?
func (a *Client) CopyFromRemote(file *os.File, remotePath string) error {
	return a.CopyFromRemotePassThru(file, remotePath, nil)
}

// Copy a file from the remote to the given writer. The passThru parameter can be used
// to keep track of progress and how many bytes that were download from the remote.
// `passThru` can be set to nil to disable this behaviour.
func (a *Client) CopyFromRemotePassThru(w io.Writer, remotePath string, passThru PassThru) error {
	wg := sync.WaitGroup{}
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		var err error

		defer func() {
			if err != nil {
				errCh <- err
			}
			errCh <- err
			wg.Done()
		}()

		r, err := a.Session.StdoutPipe()
		if err != nil {
			errCh <- err
			return
		}

		in, err := a.Session.StdinPipe()
		if err != nil {
			errCh <- err
			return
		}
		defer in.Close()

		err = a.Session.Start(fmt.Sprintf("%s -f %q", a.RemoteBinary, remotePath))
		if err != nil {
			errCh <- err
			return
		}

		err = Ack(in)
		if err != nil {
			errCh <- err
			return
		}

		res, err := ParseResponse(r)
		if err != nil {
			errCh <- err
			return
		}

		infos, err := res.ParseFileInfos()
		if err != nil {
			errCh <- err
			return
		}

		err = Ack(in)
		if err != nil {
			errCh <- err
			return
		}

		if passThru != nil {
			r = passThru(r, infos.Size)
		}

		_, err = CopyN(w, r, infos.Size)
		if err != nil {
			errCh <- err
			return
		}

		err = Ack(in)
		if err != nil {
			errCh <- err
			return
		}

		err = a.Session.Wait()
		if err != nil {
			errCh <- err
			return
		}
	}()

	if waitTimeout(&wg, a.Timeout) {
		return errors.New("timeout when download files")
	}

	close(errCh)
	return <-errCh
}

func (a *Client) Close() {
	if a.Session != nil {
		a.Session.Close()
	}
	if a.Conn != nil {
		a.Conn.Close()
	}
}
