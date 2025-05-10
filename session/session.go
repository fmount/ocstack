package session

import "github.com/ollama/ollama/api"

type Session struct {
	Client  *api.Client
	Model   string
	History History
	Tools   []api.Tool
}

type History struct {
	Text []api.Message
}
