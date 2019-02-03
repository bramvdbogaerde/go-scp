package scp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Command represents a SCP command sent to or from the remote system
type Command struct {
	Permissions os.FileMode
	Size        uint64
	Filename    string
}

func NewCommand(permissions, filename string, size uint64) (*Command, error) {
	p, err := strconv.ParseInt(permissions, 8, 64)
	if err != nil {
		return nil, err
	}

	return &Command{
		Permissions: os.FileMode(p),
		Filename:    filename,
		Size:        size,
	}, nil
}

// MarshalText implements the TextMarshaler interface
func (c *Command) MarshalText() (text []byte, err error) {
	if c.Permissions > os.ModePerm {
		return nil, fmt.Errorf("bad permissions %o (0%d)", c.Permissions, c.Permissions)
	}
	perm := strconv.FormatInt(int64(c.Permissions), 8)

	return []byte(fmt.Sprintf("C0%s %d %s", perm, c.Size, c.Filename)), nil
}

// UnmarshalText implements the TextUnmarshaler interface
func (c *Command) UnmarshalText(text []byte) error {
	cmd := string(text)
	parts := strings.Split(strings.Trim(cmd, "\n\x00"), " ")

	if len(parts) != 3 {
		return fmt.Errorf("Command '%s' is invalid", text)
	}

	perms, err := strconv.ParseInt(parts[0][1:], 8, 64)
	if err != nil {
		return err
	}

	size, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	*c = Command{
		Permissions: os.FileMode(perms),
		Size:        uint64(size),
		Filename:    parts[2],
	}

	return nil
}
