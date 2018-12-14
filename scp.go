/* Copyright (c) 2018 Bram Vandenbogaerde
 * You may use, distribute or modify this code under the
 * terms of the Mozilla Public License 2.0, which is distributed
 * along with the source code.
 */

// Simple scp package to copy files over SSH
package scp

import (
	"golang.org/x/crypto/ssh"
	"time"
)

// Returns a new scp.Client with provided host and ssh.clientConfig
// It has a default timeout of one minute.
func NewClient(host string, config *ssh.ClientConfig) Client {
	return NewClientWithTimeout(host, config, time.Minute)
}

// Returns a new scp.Client with provides host, ssh.ClientConfig and timeout
func NewClientWithTimeout(host string, config *ssh.ClientConfig, timeout time.Duration) Client {
	return Client{
		Host:         host,
		ClientConfig: config,
		Timeout:      timeout,
	}
}
