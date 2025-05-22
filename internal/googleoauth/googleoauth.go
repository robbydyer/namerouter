package googleoauth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var config = &oauth2.Config{
	RedirectURL: "https://auth.robbydyer.com/",
	Endpoint:    google.Endpoint,
}
