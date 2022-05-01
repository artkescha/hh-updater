package server

import (
	"github.com/artkescha/hh-updater/hhclient"
	"strings"
)

func upExperience(companies []hhclient.Company, prefix string) {
	for idx, _ := range companies {
		companies[idx].Name = updateName(companies[idx].Name, prefix)
	}
}

func updateName(name string, suffix string) string {
	if strings.HasSuffix(name, suffix) {
		return strings.TrimSuffix(name, suffix)
	}
	return name + suffix
}
