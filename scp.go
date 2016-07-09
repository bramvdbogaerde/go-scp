// Simple scp package to copy files over SSH
package scp

import (
	"golang.org/x/crypto/ssh"
)

// Returns a new scp.Client with provided host and ssh.clientConfig
func NewClient(host string, config *ssh.ClientConfig) Client {
	return Client{
		Host:         host,
		ClientConfig: config,
	}
}
