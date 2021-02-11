package container

import (
	"fmt"
	"github.com/syndtr/gocapability/capability"
	"strings"
)

func DefaultCapabilities() ([]capability.Cap, error) {
	return capSlice([]string{
		"CAP_CHOWN",
		"CAP_DAC_OVERRIDE",
		"CAP_FSETID",
		"CAP_FOWNER",
		"CAP_MKNOD",
		"CAP_NET_RAW",
		"CAP_SETGID",
		"CAP_SETUID",
		"CAP_SETFCAP",
		"CAP_SETPCAP",
		"CAP_NET_BIND_SERVICE",
		"CAP_SYS_CHROOT",
		"CAP_KILL",
		"CAP_AUDIT_WRITE",
	})
}

var capabilityMap map[string]capability.Cap

func init() {
	capabilityMap = make(map[string]capability.Cap, capability.CAP_LAST_CAP+1)
	for _, c := range capability.List() {
		if c > capability.CAP_LAST_CAP {
			continue
		}
		capabilityMap["CAP_"+strings.ToUpper(c.String())] = c
	}
}

func capSlice(caps []string) ([]capability.Cap, error) {
	out := make([]capability.Cap, len(caps))
	for i, c := range caps {
		v, ok := capabilityMap[c]
		if !ok {
			return nil, fmt.Errorf("unknown capability %q", c)
		}
		out[i] = v
	}
	return out, nil
}
