package server

import (
	"github.com/artkescha/hh-updater/hhclient"
	"strings"
)

func updateCompanyName(resume *hhclient.Resume, suffix string) {
	for index, company := range resume.Experience {
		resume.Experience[index].Name = addSuffix(company.Name, suffix)
	}
}

func addSuffix(old string, suffix string) string {
	if strings.HasSuffix(old, suffix) {
		return strings.TrimSuffix(old, suffix)
	}
	old += suffix
	return old
}
