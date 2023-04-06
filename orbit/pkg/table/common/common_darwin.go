//go:build darwin
// +build darwin

package common

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/antchfx/xmlquery"
)

// GetConsoleUidGid gets the uid and gid of the current (or more accurately, most recently logged
// in) *console* user. In most scenarios this should be the currently logged in user on the system.
// Note that getting the current user of the Orbit process is typically going to return root and we
// need the underlying user.
func GetConsoleUidGid() (uid uint32, gid uint32, err error) {
	info, err := os.Stat("/dev/console")
	if err != nil {
		return 0, 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected type %T", info.Sys())
	}
	return stat.Uid, stat.Gid, nil
}

// GetValFromXMLWithTags Will look for a sequence of tags and will get the following nested value as string
// In the following xml example the function will return "5" if called with parentTag = "someParentTag", tag = "someTag", tagValue = "someValue", valType = "integer"
// <someParentTag>
//
//	<someTag>someValue</someTag>
//	<integer>5</integer>
//
// </someParentTag>
func GetValFromXMLWithTags(xml string, parentTag string, tag string, tagValue string, valType string) (maxFailedAttempts string, err error) {
	doc, err := xmlquery.Parse(strings.NewReader(xml))
	if err != nil {
		return "", errors.New("can't parse xml")
	}

	for _, channel := range xmlquery.Find(doc, "//"+parentTag) {
		if n := channel.SelectElement(tag); n != nil {
			if n.InnerText() != tagValue {
				continue
			}
		}
		if n := channel.SelectElement(valType); n != nil {
			return n.InnerText(), nil
		}
	}
	return "", errors.New("can't find requested value")
}
