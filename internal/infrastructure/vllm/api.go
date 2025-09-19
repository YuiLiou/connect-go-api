package vllm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type VLLMAPI struct {
	Endpoint string
}

func (a *VLLMAPI) callAPI(action, model string) error {
	var url string
	switch action {
	case "start":
		url = fmt.Sprintf("%s/v1/start", a.Endpoint)
	case "stop":
		url = fmt.Sprintf("%s/v1/stop", a.Endpoint)
	case "update":
		url = fmt.Sprintf("%s/v1/update", a.Endpoint)
	default:
		return fmt.Errorf("invalid action: %s", action)
	}
	payload, _ := json.Marshal(map[string]string{"model": model})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vLLM API error: %s", resp.Status)
	}
	return nil
}

func (a *VLLMAPI) Start(model string) error  { return a.callAPI("start", model) }
func (a *VLLMAPI) Stop(model string) error   { return a.callAPI("stop", model) }
func (a *VLLMAPI) Update(model string) error { return a.callAPI("update", model) }
