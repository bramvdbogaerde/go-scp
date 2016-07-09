package auth

import (
	"golang.org/x/crypto/ssh"
	"io/ioutil"
)

// Loads a private and public key from "path" and returns a SSH ClientConfig to authenticate with the server
func PrivateKey(username string, path string) (ssh.ClientConfig, error) {
	private_key, err := ioutil.ReadFile(path)

	if err != nil {
		return ssh.ClientConfig{}, err
	}

	signer, err := ssh.ParsePrivateKey(private_key)

	if err != nil {
		return ssh.ClientConfig{}, err
	}

	return ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}, nil
}
