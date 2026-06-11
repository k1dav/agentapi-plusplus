// Package main is a minimal Go client that drives an AgentAPI++ server.
//
// Usage:
//
//	go run ./examples/client-go/ http://localhost:3284 "summarise this repo"
//
// It dials the server, prints the agent's status, sends the given
// user message, then prints the most recent assistant message.
// Uses the github.com/coder/agentapi-sdk-go module, which is the
// official OpenAPI-generated client for the server in this repo.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	agentapisdk "github.com/coder/agentapi-sdk-go"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: client-go <base-url> <message>")
		os.Exit(2)
	}
	baseURL, prompt := os.Args[1], os.Args[2]

	httpClient := &http.Client{Timeout: 30 * time.Second}
	client, err := agentapisdk.NewClient(baseURL, agentapisdk.WithHTTPClient(httpClient))
	if err != nil {
		log.Fatalf("dial %s: %v", baseURL, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	status, err := client.GetStatus(ctx)
	if err != nil {
		log.Fatalf("status: %v", err)
	}
	fmt.Printf("status=%s\n", status.Status)

	// PostMessage accepts a MessageRequestBody; the SDK exports the
	// message-type constants MessageTypeUser / MessageTypeRaw.
	if _, err := client.PostMessage(ctx, agentapisdk.PostMessageParams{
		Type:    agentapisdk.MessageTypeUser,
		Content: prompt,
	}); err != nil {
		log.Fatalf("send: %v", err)
	}

	// Poll until the agent is stable again. A production client would
	// subscribe to /events instead; this is the smallest possible
	// loop and is fine for short prompts.
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		st, err := client.GetStatus(ctx)
		if err != nil {
			log.Fatalf("status poll: %v", err)
		}
		if st.Status == agentapisdk.StatusStable {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	msgs, err := client.GetMessages(ctx)
	if err != nil {
		log.Fatalf("messages: %v", err)
	}
	if n := len(msgs.Messages); n > 0 {
		fmt.Println("--- last message ---")
		fmt.Println(msgs.Messages[n-1].Content)
	}
}
