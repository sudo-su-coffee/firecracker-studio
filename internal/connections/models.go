package connections

import "time"

type Account struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type Server struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Kind        string     `json:"kind"`
	Username    string     `json:"username,omitempty"`
	Health      string     `json:"health"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
	Active      bool       `json:"active"`
}
