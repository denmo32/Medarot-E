package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	resource "github.com/quasilyte/ebitengine-resource" // resource パッケージを追加
)

var placeholderRegex = regexp.MustCompile(`{(\w+)}`)

// MessageManager handles loading and retrieving formatted messages.
type MessageManager struct {
	messages map[string]string
	loader   *resource.Loader // resource.Loader を追加
}

// NewMessageManager creates and initializes a new MessageManager.
func NewMessageManager(filePath string, loader *resource.Loader) (*MessageManager, error) {
	mm := &MessageManager{
		messages: make(map[string]string),
		loader:   loader, // loader を設定
	}
	err := mm.LoadMessages(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}
	return mm, nil
}

// LoadMessages loads message templates from a JSON file using the resource loader.
func (mm *MessageManager) LoadMessages(filePath string) error {
	// resource loader を使用してメッセージデータをロード
	// filePath は assetPaths.Messages から来るので、RawMessagesJSON を使用
	res := mm.loader.LoadRaw(RawMessagesJSON)
	if res.Data == nil { // res.Data が nil かどうかでチェック
		return fmt.Errorf("could not load message resource %s: data is nil", filePath)
	}

	var templates []MessageTemplate
	err := json.Unmarshal(res.Data, &templates)
	if err != nil {
		return fmt.Errorf("could not parse message data from %s: %w", filePath, err)
	}

	for _, t := range templates {
		mm.messages[t.ID] = t.Text
	}
	log.Printf("Loaded %d messages from %s", len(mm.messages), filePath)
	return nil
}

// GetMessage retrieves a raw message template by its ID.
func (mm *MessageManager) GetRawMessage(id string) (string, bool) {
	msg, found := mm.messages[id]
	return msg, found
}

// FormatMessage formats a message template with the given parameters.
// It handles two types of placeholders:
// 1. {key} - replaced by params[key]
// 2. %s, %d, %f - standard fmt.Sprintf style, using ordered args from params["ordered_args"]
func (mm *MessageManager) FormatMessage(id string, params map[string]interface{}) string {
	template, ok := mm.messages[id]
	if !ok {
		log.Printf("Warning: Message with ID '%s' not found.", id)
		return id // Return ID if not found, so it's noticeable
	}

	// Handle ordered arguments for fmt.Sprintf style placeholders
	if orderedArgs, ok := params["ordered_args"].([]interface{}); ok {
		// Count standard fmt specifiers to avoid errors if not enough args are provided.
		// This is a simplified count; a more robust solution might involve detailed parsing.
		numSpecifiers := strings.Count(template, "%s") +
			strings.Count(template, "%d") +
			strings.Count(template, "%f") +
			strings.Count(template, "%v") // Add other specifiers as needed

		if len(orderedArgs) < numSpecifiers {
			log.Printf("Warning: Not enough ordered_args for message ID '%s'. Expected %d, got %d. Template: %s", id, numSpecifiers, len(orderedArgs), template)
			// Attempt to format with available args, or return template to avoid panic
			if len(orderedArgs) > 0 {
				return fmt.Sprintf(template, orderedArgs[:min(len(orderedArgs), numSpecifiers)]...)
			}
			return template // Not enough args to even attempt formatting safely
		}
		// Ensure we don't pass more args than there are specifiers, if that could be an issue.
		// fmt.Sprintf is generally okay with extra args if they are not consumed by specifiers.
		return fmt.Sprintf(template, orderedArgs...)
	}

	// Handle {key} style placeholders
	formattedMessage := placeholderRegex.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		if val, pOk := params[key]; pOk {
			return fmt.Sprintf("%v", val)
		}
		log.Printf("Warning: Placeholder '%s' not found in params for message ID '%s'", match, id)
		return match // Return the placeholder itself if key not in params
	})

	return formattedMessage
}

// Helper function, similar to math.Min but for integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
