package hhclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type ResumeService service

type ResumeList struct {
	Resumes []*Resume `json:"items"`
	Page    int       `json:"page"`
	PerPage int       `json:"per_page"`
	Pages   int       `json:"pages"`
	Found   int       `json:"found"`
}

type Resume struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	TotalViews    int       `json:"total_views"`
	NewViews      int       `json:"new_views"`
	URL           string    `json:"url"`
	LastName      string    `json:"last_name"`
	FirstName     string    `json:"first_name"`
	MiddleName    string    `json:"middle_name"`
	Age           int       `json:"age"`
	Experience    []Company `json:"experience"`
	NextPublishAt string    `json:"next_publish_at"`
}

type Company struct {
	Name        string `json:"company"`
	Position    string `json:"position"`
	Start       string `json:"start"`
	Description string `json:"description"`
}

type ResumeStatus struct {
	Blocked            bool   `json:"blocked"`
	Finished           bool   `json:"finished"`
	CanPublishOrUpdate bool   `json:"can_publish_or_update"`
	PublishURL         string `json:"publish_url"`
}

func (r *ResumeService) ResumeMine() ([]*Resume, error) {
	resp, err := r.client.Get(DefaultBaseURL + "resumes/mine")
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
	if code := resp.StatusCode; code < 200 || code > 299 {
		return nil, fmt.Errorf("Incorrect status code (%s)", resp.Status)
	}
	var resumeList ResumeList
	if err := json.Unmarshal(body, &resumeList); err != nil {
		return nil, err
	}
	return resumeList.Resumes, nil
}

func (r *ResumeService) ResumePublish(resume *Resume) error {
	resp, err := r.client.Post(fmt.Sprintf("%sresumes/%s/publish", DefaultBaseURL, resume.ID), "", nil)
	if err != nil {
		return err
	}
	if code := resp.StatusCode; code < 200 || code > 299 {
		logrus.Debug("resumes publish fail status: %d", code)
		return fmt.Errorf("Incorrect status code (%s)", resp.Status)
	}
	return nil
}

func (r *ResumeService) EditResume(resume *Resume) error {
	payload, err := json.Marshal(&resume)
	if err != nil {
		return fmt.Errorf("resume with title unmarshal fail %s", err)
	}
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%sresumes/%s", DefaultBaseURL, resume.ID), bytes.NewBuffer(payload))
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("do reqest fail %s", err)
	}
	if code := resp.StatusCode; code < 200 || code > 299 {
		return fmt.Errorf("Incorrect status code (%s)", resp.Status)
	}
	return nil
}

func (r *ResumeService) ReadResume(resumeId string) (*Resume, error) {
	resp, err := r.client.Get(fmt.Sprintf("%sresumes/%s", DefaultBaseURL, resumeId))
	if err != nil {
		return nil, err
	}
	if code := resp.StatusCode; code < 200 || code > 299 {
		return nil, fmt.Errorf("Incorrect status code (%s)", resp.Status)
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
	var resume Resume
	if err := json.Unmarshal(body, &resume); err != nil {
		return nil, err
	}
	return &resume, nil
}

func (r *ResumeService) ResumesStatus(resume *Resume) (*ResumeStatus, error) {
	resp, err := r.client.Get(fmt.Sprintf("%sresumes/%s/status", DefaultBaseURL, resume.ID))
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
	if code := resp.StatusCode; code < 200 || code > 299 {
		logrus.Debug("resumes status: %v", code)
		return nil, fmt.Errorf("Incorrect status code (%s)", resp.Status)
	}
	var resumeStatus *ResumeStatus
	if err := json.Unmarshal(body, &resumeStatus); err != nil {
		return nil, err
	}
	return resumeStatus, nil
}
