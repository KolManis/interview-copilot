package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/KolManis/interview-copilot/internal/ai"
	"github.com/KolManis/interview-copilot/internal/audio"
	"github.com/joho/godotenv"
)

type Copilot struct {
	groq     *ai.GroqClient
	recorder *audio.Recorder
	history  []string
}

func NewCopilot() *Copilot {
	godotenv.Load()
	apiKey := os.Getenv("GROQ_API_KEY")

	return &Copilot{
		groq:     ai.NewGroqClient(apiKey),
		recorder: audio.NewRecorder(),
		history:  []string{},
	}
}

func (c *Copilot) HandleVoiceInput() {
	audioFile, err := c.recorder.Record(5)
	if err != nil {
		fmt.Printf("❌ Ошибка записи: %v\n", err)
		return
	}
	defer ai.Cleanup(audioFile)

	fmt.Print("📝 Распознаю речь... ")
	text, err := c.groq.TranscribeAudio(audioFile)
	if err != nil {
		fmt.Printf("\n❌ Ошибка распознавания: %v\n", err)
		return
	}

	fmt.Printf("\n💬 Вы сказали: %s\n", text)

	fmt.Print("🤔 Генерирую подсказку... ")
	answer, err := c.groq.AskQuestion(text, c.history)
	if err != nil {
		fmt.Printf("\n❌ Ошибка: %v\n", err)
		return
	}

	c.history = append(c.history, fmt.Sprintf("Q: %s", text))
	c.history = append(c.history, fmt.Sprintf("A: %s", answer))

	fmt.Printf("\n✅ Подсказка:\n%s\n", answer)
}

func (c *Copilot) HandleTextInput(question string) {
	fmt.Print("🤔 Думаю... ")

	answer, err := c.groq.AskQuestion(question, c.history)
	if err != nil {
		fmt.Printf("\n❌ Ошибка: %v\n", err)
		return
	}

	c.history = append(c.history, fmt.Sprintf("Q: %s", question))
	c.history = append(c.history, fmt.Sprintf("A: %s", answer))

	fmt.Printf("\n✅ Подсказка:\n%s\n", answer)
}

func main() {
	copilot := NewCopilot()
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🎤 AI Собеседовательный Копайлот (Groq)")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Команды:")
	fmt.Println("  /listen     - голосовой вопрос (5 сек запись)")
	fmt.Println("  /h          - история диалога")
	fmt.Println("  /c          - очистить историю")
	fmt.Println("  /quit       - выход")
	fmt.Println("  <вопрос>    - текстовый вопрос")
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

		case input == "/listen":
			copilot.HandleVoiceInput()

		case input == "/h":
			if len(copilot.history) == 0 {
				fmt.Println("📝 История пуста")
				continue
			}
			fmt.Println("\n📝 История диалога:")
			for i, msg := range copilot.history {
				fmt.Printf("%d. %s\n", i+1, msg)
			}

		case input == "/c":
			copilot.history = []string{}
			fmt.Println("🧹 История очищена")

		case input != "":
			copilot.HandleTextInput(input)
		}
	}
}
