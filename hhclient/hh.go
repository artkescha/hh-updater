package hhclient

import (
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

const (
	DefaultBaseURL = "https://api.hh.ru/"
)

type Client struct {
	BaseURL   *url.URL
	UserAgent string

	Me     *MeService
	Resume *ResumeService
}

type service struct {
	client *http.Client
}

func NewClient(token *oauth2.Token) *Client {
	httpClient := &http.Client{
		Transport: &TokenTransport{
			AccessToken: token.AccessToken,
		},
	}
	c := &Client{}
	c.Me = &MeService{httpClient}
	c.Resume = &ResumeService{httpClient}
	return c
}
