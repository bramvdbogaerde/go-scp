package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jimrosenfeld/go-scp"
	"github.com/jimrosenfeld/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

const (
	TestUsername       = "testuser"
	TestKeyUsername    = "testuserkey"
	TestPassword       = "testpass"
	TestHostname       = "localhost:2200"
	TestFilename       = "transfer-file"
	TestPrivateKeyName = "test_rsa"
)

func main() {
	fmt.Println("copying file with scp...")
	copyFile()

	fmt.Println("copying file with scp using sudo with key...")
	copyFileSudoKey()

	fmt.Println("copying file with scp using sudo with password...")
	copyFileSudo()
}

func copyFile() {
	clientConfig, err := auth.PasswordKey(TestUsername, TestPassword, ssh.InsecureIgnoreHostKey())
	if err != nil {
		log.Fatalln("can't get clientConfig:", err)
	}

	client := scp.NewClient(TestHostname, &clientConfig)

	err = client.Connect()
	if err != nil {
		log.Fatalln("can't connect:", err)
	}

	f, err := os.Open(TestFilename)
	if err != nil {
		log.Fatalln("can't open file:", err)
	}

	defer f.Close()
	defer client.Close()

	remoteFilename := filepath.Join("/home/testuser", TestFilename)

	err = client.CopyFile(f, remoteFilename, "0660")
	if err != nil {
		log.Fatalln("can't copy file:", err)
	}

	fmt.Println("copied", TestFilename, "to", remoteFilename)
}

func copyFileSudoKey() {
	clientConfig, err := auth.PrivateKey(TestKeyUsername, TestPrivateKeyName, ssh.InsecureIgnoreHostKey())
	if err != nil {
		log.Fatalln("can't get clientConfig:", err)
	}

	client := scp.NewClient(TestHostname, &clientConfig)
	client.RemoteBinary = "sudo scp"

	err = client.Connect()
	if err != nil {
		log.Fatalln("can't connect:", err)
	}

	f, err := os.Open(TestFilename)
	if err != nil {
		log.Fatalln("can't open file:", err)
	}

	defer f.Close()
	defer client.Close()

	remoteFilename := filepath.Join("/root", TestFilename)

	err = client.CopyFile(f, remoteFilename, "0660")
	if err != nil {
		log.Fatalln("can't copy file:", err)
	}

	fmt.Println("copied", TestFilename, "to", remoteFilename)
}

func copyFileSudo() {
	clientConfig, err := auth.PasswordKey(TestUsername, TestPassword, ssh.InsecureIgnoreHostKey())
	if err != nil {
		log.Fatalln("can't get clientConfig:", err)
	}

	client := scp.NewClientWithSudoPassword(TestHostname, &clientConfig, TestPassword)

	err = client.Connect()
	if err != nil {
		log.Fatalln("can't connect:", err)
	}

	f, err := os.Open(TestFilename)
	if err != nil {
		log.Fatalln("can't open file:", err)
	}

	defer f.Close()
	defer client.Close()

	remoteFilename := filepath.Join("/root", TestFilename) + "-sudo"

	err = client.CopyFile(f, remoteFilename, "0660")
	if err != nil {
		log.Fatalln("can't copy file:", err)
	}

	fmt.Println("copied", TestFilename, "to", remoteFilename)
}
