package scp

import (
	"fmt"
	"os"
	"testing"
)

func TestCommand_Marshal(t *testing.T) {
	tests := map[string]struct {
		perms  os.FileMode
		size   uint
		name   string
		assert string
		err    error
	}{
		"full permissions": {
			perms:  os.ModePerm,
			size:   42,
			name:   "test",
			assert: "C0777 42 test",
		},
		"bad permissions": {
			perms: 21312,
			size:  42,
			name:  "test",
			err:   fmt.Errorf("bad permissions %o (0%d)", 21312, 21312),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := Command{
				Permissions: tc.perms,
				Size:        tc.size,
				Filename:    tc.name,
			}

			ct, err := c.MarshalText()
			checkErr(t, err, tc.err)

			if string(ct) != tc.assert {
				t.Errorf("%s != %s", ct, tc.assert)
			}
		})
	}
}

func TestCommand_Unarshal(t *testing.T) {
	tests := map[string]struct {
		text   string
		assert Command
		err    error
	}{
		"full permissions": {
			text: "C0777 42 test",
			assert: Command{
				Permissions: os.ModePerm,
				Size:        42,
				Filename:    "test",
			},
		},
		"invalid command": {
			text: "C0777 42 test extra",
			err:  fmt.Errorf("Command 'C0777 42 test extra' is invalid"),
		},
		"bad permissions command": {
			text: "C2853 42 test",
			err:  fmt.Errorf("strconv.ParseInt: parsing \"2853\": invalid syntax"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Command{}
			err := c.UnmarshalText([]byte(tc.text))
			if checkErr(t, err, tc.err) {
				return
			}

		})
	}
}

func checkErr(t *testing.T, err, caseErr error) (checkedErr bool) {
	if err != nil && caseErr == nil {
		t.Errorf("failed with error: %s", err)
	}
	if err != nil && caseErr != nil {
		if err.Error() != caseErr.Error() {
			t.Errorf("unmatched errors: %s != %s", err, caseErr)
		}

		return true
	}

	return false
}

func TestCommand_New(t *testing.T) {
	tests := map[string]struct {
		perms  string
		size   uint
		name   string
		assert string
		err    error
	}{
		"full permissions": {
			perms:  "777",
			size:   42,
			name:   "test",
			assert: "C0777 42 test",
		},
		"bad parse": {
			perms: "779",
			size:  42,
			name:  "test",
			err:   fmt.Errorf("strconv.ParseInt: parsing \"779\": invalid syntax"),
		},
		"bad permissions": {
			perms: "21323",
			size:  42,
			name:  "test",
			err:   fmt.Errorf("bad permissions 21323 (08915)"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c, err := NewCommand(tc.perms, tc.name, tc.size)
			if checkErr(t, err, tc.err) {
				return
			}

			ct, err := c.MarshalText()
			checkErr(t, err, tc.err)

			if string(ct) != tc.assert {
				t.Errorf("%s != %s", ct, tc.assert)
			}
		})
	}
}
