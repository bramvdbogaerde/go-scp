package scp

import (
	"golang.org/x/crypto/ssh"
)

func NewClient(host string, config *ssh.ClientConfig) Client {
	return Client{
		Host:         host,
		ClientConfig: config,
	}
}
