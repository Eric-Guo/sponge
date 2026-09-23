// Package gemini provides a client for the Google generative AI API.
package gemini

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/Eric-Guo/sponge/pkg/aicli"
)

// Client is a Google generative AI client.
type Client struct {
	apiKey    string
	ModelName string
	Cli       *genai.Client
	Model     *genai.GenerativeModel

	enableContext   bool              // whether to use assistant context, default is false
	contextMessages []*ContextMessage // assistant context
}

// ContextMessage chat history message
type ContextMessage struct {
	Role    string `json:"role"` // "user" or "model"
	Content string `json:"content"`
}

// NewClient creates a new Google generative AI client.
func NewClient(apiKey string, opts ...ClientOption) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("API key cannot be empty")
	}

	c := defaultClientOptions()
	c.apply(opts...)
	if c.ModelName == "" {
		c.ModelName = DefaultModel
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	model := client.GenerativeModel(c.ModelName)

	return &Client{
		apiKey:          apiKey,
		ModelName:       c.ModelName,
		Cli:             client,
		Model:           model,
		enableContext:   c.enableContext,
		contextMessages: c.contextMessages,
	}, nil
}

// Close closes the client.
func (c *Client) Close() error {
	return c.Cli.Close()
}

// Send sends a prompt to the gemini model and returns the response.
func (c *Client) Send(ctx context.Context, prompt string, files ...string) (string, error) {
	if prompt == "" {
		return "", errors.New("prompt cannot be empty")
	}
	parts, err := c.setMessages(prompt, files...)
	if err != nil {
		return "", err
	}

	resp, err := c.Model.GenerateContent(ctx, parts...)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	replyContent := ""
	for _, candidate := range resp.Candidates {
		if len(candidate.Content.Parts) > 0 {
			part, ok := candidate.Content.Parts[0].(genai.Text)
			if !ok {
				continue
			}
			replyContent += string(part)
		}
	}
	if replyContent == "" {
		return "", errors.New("reply content is empty")
	}
	c.appendAssistantContext(prompt, replyContent)

	return replyContent, nil
}

// SendStream sends a prompt to the gemini model and returns a channel of responses.
func (c *Client) SendStream(ctx context.Context, prompt string, files ...string) *aicli.StreamReply {
	response := &aicli.StreamReply{Content: make(chan string), Err: error(nil)}

	go func() {
		defer func() { close(response.Content) }()
		if prompt == "" {
			response.Err = errors.New("prompt cannot be empty")
			return
		}
		parts, err := c.setMessages(prompt, files...)
		if err != nil {
			response.Err = err
			return
		}

		var replyContent string
		defer func() {
			if response.Err == nil && replyContent != "" {
				c.appendAssistantContext(prompt, replyContent)
			}
		}()

		stream := c.Model.GenerateContentStream(ctx, parts...)
		for {
			resp, err := stream.Next()
			if err != nil {
				if errors.Is(err, iterator.Done) {
					break
				}
				response.Err = fmt.Errorf("stream.Next() error: %w", err)
				return
			}
			for _, candidate := range resp.Candidates {
				for _, part := range candidate.Content.Parts {
					partBytes, ok := part.(genai.Text)
					if !ok {
						continue
					}
					partText := string(partBytes)
					select {
					case <-ctx.Done():
						response.Err = ctx.Err()
						return
					case response.Content <- partText:
						replyContent += partText
					}
				}
			}
		}
	}()

	return response
}

// ListModelNames lists the available models.
func (c *Client) ListModelNames(ctx context.Context) ([]string, error) {
	var modelNames []string

	iter := c.Cli.ListModels(ctx)
	for {
		model, err := iter.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return modelNames, fmt.Errorf("error retrieving model: %v", err)
		}
		modelNames = append(modelNames, model.Name)
	}

	for i, name := range modelNames {
		modelNames[i] = strings.TrimPrefix(name, "models/")
	}

	return modelNames, nil
}

// ListContextMessages list assistant context messages
func (c *Client) ListContextMessages() []*ContextMessage {
	return c.contextMessages
}

// RefreshContext refreshes assistant context
func (c *Client) RefreshContext() {
	if len(c.contextMessages) > 0 {
		c.contextMessages = []*ContextMessage{}
	}
}

func (c *Client) appendAssistantContext(prompt string, replyContent string) {
	if c.enableContext && replyContent != "" {
		c.contextMessages = append(c.contextMessages, &ContextMessage{Role: RoleUser, Content: prompt})
		c.contextMessages = append(c.contextMessages, &ContextMessage{Role: RoleModel, Content: replyContent})
	}
}

func (c *Client) setMessages(prompt string, files ...string) ([]genai.Part, error) {
	var parts []genai.Part

	// history context
	for _, msg := range c.contextMessages {
		parts = append(parts, genai.Text(fmt.Sprintf("%s: %s", msg.Role, msg.Content)))
	}

	// file message
	if len(files) > 0 {
		for _, file := range files {
			mimeType := mime.TypeByExtension(filepath.Ext(file))
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			parts = append(parts, genai.Blob{
				MIMEType: mimeType,
				Data:     data,
			})
		}
	}

	// user message
	parts = append(parts, genai.Text(prompt))

	return parts, nil
}
