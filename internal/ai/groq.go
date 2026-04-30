package ai

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type GroqClient struct {
	apiKey string
	chat   *openai.Client
}

func NewGroqClient(apiKey string) *GroqClient {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1"

	return &GroqClient{
		apiKey: apiKey,
		chat:   openai.NewClientWithConfig(config),
	}
}

func (g *GroqClient) TranscribeAudio(audioPath string) (string, error) {
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return "", fmt.Errorf("аудио файл не найден: %s", audioPath)
	}

	resp, err := g.chat.CreateTranscription(
		context.Background(),
		openai.AudioRequest{
			Model:    "whisper-large-v3",
			FilePath: audioPath,
			Language: "ru",
			// Подсказки для улучшения распознавания технических терминов
			Prompt: "Go, Golang, Python, Java, Kotlin, Docker, Kubernetes, " +
				"горутины, интерфейсы, каналы, микросервисы, " +
				"собеседование, архитектура, алгоритм",
		},
	)

	if err != nil {
		return "", fmt.Errorf("ошибка транскрибации: %v", err)
	}

	return resp.Text, nil
}

func (g *GroqClient) AskQuestion(question string, history []string) (string, error) {
	systemPrompt := `Ты - ассистент на техническом собеседовании.
Давай краткие подсказки по делу.
Отвечай на русском языке.
Структура ответа:
- Если вопрос про технологии: 2-3 ключевых пункта
- Если Behavioral: структура ответа по STAR
- Если про код: покажи решение с коротким комментарием
Не пиши "вот ответ", сразу давай подсказку.`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	for _, msg := range history {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: msg,
		})
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: question,
	})

	resp, err := g.chat.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       "llama-3.3-70b-versatile",
			Messages:    messages,
			Temperature: 0.7,
			MaxTokens:   300,
		},
	)

	if err != nil {
		return "", fmt.Errorf("ошибка запроса: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func Cleanup(audioPath string) {
	os.Remove(audioPath)
}
