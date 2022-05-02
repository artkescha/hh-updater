package server

import (
	"github.com/artkescha/hh-updater/hhclient"
	"strings"
)

func upExperience(companies []hhclient.Company, prefix string) {
	for idx, _ := range companies {
		companies[idx].Description = updateDescription(companies[idx].Description, prefix)
	}
}

func updateDescription(name string, suffix string) string {
	if strings.HasSuffix(name, suffix) {
		return strings.TrimSuffix(name, suffix)
	}
	return name + suffix
}
