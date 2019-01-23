package scp

import (
	"fmt"
	"strconv"
	"strings"
)

// Command represents a SCP command sent to or from the remote system
type Command struct {
	Permissions int
	Size        uint
	Filename    string
}

// MarshalText implements the TextMarshaler interface
func (c *Command) MarshalText() (text []byte, err error) {
	perms := strconv.Itoa(c.Permissions)
	size := strconv.Itoa(int(c.Size))

	return []byte(fmt.Sprintf("C%s %s %s", perms, size, c.Filename)), nil
}

// UnmarshalText implements the TextUnmarshaler interface
func (c *Command) UnmarshalText(text []byte) error {
	cmd := string(text)
	parts := strings.Split(strings.Trim(cmd, "\n\x00"), " ")

	if len(parts) != 3 {
		return fmt.Errorf("Command '%s' invalid", cmd)
	}

	perms, err := strconv.Atoi(parts[0][1:])
	if err != nil {
		return err
	}

	size, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	*c = Command{
		Permissions: perms,
		Size:        uint(size),
		Filename:    parts[2],
	}

	return nil
}
