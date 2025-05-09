package session

import "github.com/ollama/ollama/api"

type History struct {
	Text []api.Message
}
