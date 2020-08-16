// Package hh provides constants for using OAuth2 to access HH.
package hh // import "github.com/leominov/hh"

import "golang.org/x/oauth2"

// Endpoint is HH's OAuth 2.0 endpoint.
var Endpoint = oauth2.Endpoint{
	AuthURL:  "https://hh.ru/oauth/authorize",
	TokenURL: "https://hh.ru/oauth/token",
}

func init() {
	// HH only accepts client secret in URL param
	oauth2.RegisterBrokenAuthHeaderProvider(Endpoint.TokenURL)
}
