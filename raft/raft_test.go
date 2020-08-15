package raft

import (
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type mockClient struct {
	PostFunc func(url string, contentType string, body io.Reader) (*http.Response, error)
}

func (m *mockClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	return getPostFunc(url, contentType, body)
}

var getPostFunc func(url string, contentType string, body io.Reader) (*http.Response, error)

func TestInitBecomesLeaderIfNoNodesPassedIn(t *testing.T) {
	Init("127.0.0.1:7777", "", time.Second/2, time.Second, time.Second)
	if State != "leader" {
		t.Error()
	}
	Stop()
}

func TestInitNodesInListGetIntroducedTo(t *testing.T) {
	client = &mockClient{}
	inviteCount := 0
	getPostFunc = func(url string, contentType string, body io.Reader) (*http.Response, error) {
		inviteCount++
		resp := &http.Response{Status: http.StatusText(http.StatusOK), StatusCode: http.StatusOK}
		return resp, nil
	}
	Init("127.0.0.1:7777", "127.0.0.1:7778", time.Second/2, time.Second, time.Second)
	Stop()

	if inviteCount != 1 {
		t.Error(inviteCount)
	}
}

func TestIntroduceFailsIfNodeDoesNotRespond(t *testing.T) {
	client = &mockClient{}
	inviteCount := 0
	getPostFunc = func(url string, contentType string, body io.Reader) (*http.Response, error) {
		inviteCount++
		resp := &http.Response{Status: http.StatusText(http.StatusBadRequest), StatusCode: http.StatusBadRequest}
		return resp, nil
	}
	err := introduce("127.0.0.1:7778")

	if inviteCount != 1 {
		t.Error(inviteCount)
	}

	if err == nil {
		t.Error("Introduction didn't error")
	}
}

func TestIntroduceFailsIfRequestErrors(t *testing.T) {
	client = &mockClient{}
	inviteCount := 0
	getPostFunc = func(url string, contentType string, body io.Reader) (*http.Response, error) {
		inviteCount++
		resp := &http.Response{Status: http.StatusText(http.StatusBadRequest), StatusCode: http.StatusBadRequest}
		return resp, errors.New("Whoops")
	}
	err := introduce("127.0.0.1:7778")

	if inviteCount != 1 {
		t.Error(inviteCount)
	}

	if err == nil {
		t.Error("Introduction didn't error")
	}
}
