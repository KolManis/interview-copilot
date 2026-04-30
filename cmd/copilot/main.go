package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

// еще модели
// "mixtral-8x7b-32768"
// "gemma2-9b-it"

type Copilot struct {
	client  *openai.Client
	context []string
	model   string
}

func NewCopilot() *Copilot {
	godotenv.Load()
	apiKey := os.Getenv("GROQ_API_KEY")

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1"
	client := openai.NewClientWithConfig(config)

	return &Copilot{
		client:  client,
		context: []string{},
		model:   "llama-3.3-70b-versatile",
	}
}

func (c *Copilot) AskForHelp(question string) string {
	c.context = append(c.context, fmt.Sprintf("Вопрос: %s", question))

	systemPrompt := `Ты - ассистент на техническом собеседовании.
Давай краткие подсказки по делу.
Отвечай на русском языке, даже если вопрос на английском.
Структура ответа:
- Если вопрос про технологии: 2-3 ключевых пункта
- Если Behavioral: структура ответа по STAR
- Если про код: покажи решение с коротким комментарием
Не пиши "вот ответ", сразу давай подсказку.`

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: question,
				},
			},
			Temperature: 0.7,
			MaxTokens:   300,
		},
	)

	if err != nil {
		return fmt.Sprintf("❌ Ошибка: %v", err)
	}

	answer := resp.Choices[0].Message.Content
	c.context = append(c.context, fmt.Sprintf("Ответ: %s", answer))

	return answer
}

func main() {
	copilot := NewCopilot()
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🤖 AI Собеседовательный Копайлот (Groq)")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Команды:")
	fmt.Println("  /h          - история диалога")
	fmt.Println("  /c          - очистить историю")
	fmt.Println("  /quit       - выход")
	fmt.Println("  <вопрос>    - задать вопрос")
	fmt.Println(strings.Repeat("=", 50))

	for {
		fmt.Print("\n💬 ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		switch {
		case input == "/quit":
			fmt.Println("👋 Удачи на собеседовании!")
			return

		case input == "/h":
			if len(copilot.context) == 0 {
				fmt.Println("📝 История пуста")
				continue
			}
			fmt.Println("\n📝 История диалога:")
			for i, msg := range copilot.context {
				fmt.Printf("%d. %s\n", i+1, msg)
			}

		case input == "/c":
			copilot.context = []string{}
			fmt.Println("🧹 История очищена")

		case input != "":
			fmt.Print("🤔 ... ")
			answer := copilot.AskForHelp(input)
			fmt.Printf("\n%s\n", answer)
		}
	}
}
