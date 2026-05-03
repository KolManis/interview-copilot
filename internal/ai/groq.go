package ai

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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
			Prompt:   "Go, Golang, Python, Java, Docker, Kubernetes, горутины, интерфейсы, каналы, микросервисы",
		},
	)

	if err != nil {
		return "", fmt.Errorf("ошибка транскрибации: %v", err)
	}

	return resp.Text, nil
}

// AskQuestionStream — стриминговая версия, выводит ответ по мере генерации
func (g *GroqClient) AskQuestionStream(question string, history []string) (string, error) {
	systemPrompt := `Ты - ассистент на техническом собеседовании.

ПРАВИЛА ФОРМАТИРОВАНИЯ:
1. Используй **Важно:** для выделения критических моментов
2. Используй **Пример:** перед примерами кода или сценариев
3. Используй **Ключевые моменты:** для списка главных пунктов
4. Используй **Совет:**  (обратные кавычки)

СТРУКТУРА ОТВЕТА:
- Сначала дай краткий ответ (1-2 предложения)
- Затем **Ключевые моменты:** (3-5 пунктов)
- **Пример:** с кодом или сценарием
- В конце - **Совет:** для успешного ответа на интервью

Будь полезным, конкретным и не пиши лишнего текста.`

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

	// Создаем стриминговый запрос
	stream, err := g.chat.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       "llama-3.3-70b-versatile",
			Messages:    messages,
			Temperature: 0.7,
			MaxTokens:   300,
			Stream:      true,
		},
	)
	if err != nil {
		return "", fmt.Errorf("ошибка создания стрима: %v", err)
	}
	defer stream.Close()

	// Собираем полный ответ по кусочкам
	var fullAnswer strings.Builder

	fmt.Print("\n✅ ")

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			fmt.Println() // Перевод строки в конце
			break
		}
		if err != nil {
			return "", fmt.Errorf("ошибка чтения стрима: %v", err)
		}

		if len(response.Choices) > 0 {
			chunk := response.Choices[0].Delta.Content
			fmt.Print(chunk) // Печатаем кусочек сразу
			fullAnswer.WriteString(chunk)
		}
	}

	return fullAnswer.String(), nil
}

// Обычная версия (без стриминга) — оставим для совместимости
func (g *GroqClient) AskQuestion(question string, history []string) (string, error) {
	systemPrompt := `Ты - ассистент на техническом собеседовании для Golang разработчика.

ПРАВИЛА ФОРМАТИРОВАНИЯ:
1. Используй **Важно:** для выделения критических моментов
2. Используй **Пример:** перед примерами кода или сценариев
3. Используй **Ключевые моменты:** для списка главных пунктов
4. Используй **Совет:**  (обратные кавычки)

СТРУКТУРА ОТВЕТА:
- Сначала дай краткий ответ (1-2 предложения)
- Затем **Ключевые моменты:** (3-5 пунктов)
- **Пример:** с кодом или сценарием
- В конце - **Совет:** для успешного ответа на интервью

Будь полезным, конкретным и не пиши лишнего текста.`

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
