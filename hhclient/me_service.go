package hhclient

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

type MeService service

type Me struct {
	ID          string `json:"id"`
	LastName    string `json:"last_name"`
	FirstName   string `json:"first_name"`
	MiddleName  string `json:"middle_name"`
	IsAdmin     bool   `json:"is_admin"`
	IsApplicant bool   `json:"is_applicant"`
	IsEmployer  bool   `json:"is_employer"`
	Email       string `json:"email"`
}

func (m *MeService) GetMe() (*Me, error) {
	resp, err := m.client.Get(DefaultBaseURL + "me")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.Debugf("close resp body fail %s", err)
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var me *Me
	if err := json.Unmarshal(body, &me); err != nil {
		return nil, err
	}
	return me, nil
}
