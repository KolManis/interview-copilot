// cmd/gui/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/KolManis/interview-copilot/internal/ai"
	"github.com/KolManis/interview-copilot/internal/audio"
	"github.com/joho/godotenv"
)

func init() {
	_, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Не удалось создать файл логов:", err)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	log.Println("=== ПРИЛОЖЕНИЕ ЗАПУЩЕНО ===")

	if err := godotenv.Load(); err != nil {
		log.Printf("Предупреждение: .env файл не загружен: %v", err)
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		showErrorWindow("GROQ_API_KEY не найден!\nСоздайте файл .env с вашим ключом")
		return
	}

	groqClient := ai.NewGroqClient(apiKey)
	recorder := audio.NewRecorder()

	a := app.New()
	w := a.NewWindow("AI Interview Copilot")
	w.Resize(fyne.NewSize(900, 700))

	// ========== ВЕРХНЯЯ ЧАСТЬ: поле ввода ==========
	questionEntry := widget.NewMultiLineEntry()
	questionEntry.SetPlaceHolder("Введите ваш вопрос здесь...")
	questionEntry.Wrapping = fyne.TextWrapWord

	// Ограничиваем высоту поля ввода
	questionContainer := container.NewMax(
		container.NewPadded(questionEntry),
	)

	// ========== ЦЕНТРАЛЬНАЯ ЧАСТЬ: поле ответа (ОСНОВНОЕ) ==========
	answerLabel := widget.NewLabel("")
	answerLabel.Wrapping = fyne.TextWrapWord
	answerLabel.TextStyle = fyne.TextStyle{Monospace: true}

	answerScroll := container.NewScroll(answerLabel)
	answerScroll.SetMinSize(fyne.NewSize(800, 400)) // Большое поле ответа

	// ========== НИЖНЯЯ ЧАСТЬ: кнопки и статус ==========
	var history []string

	sendButton := widget.NewButtonWithIcon("Отправить", theme.NavigateNextIcon(), func() {
		question := strings.TrimSpace(questionEntry.Text)
		if question == "" {
			return
		}

		answerLabel.SetText("🤔 Думаю...")

		go func() {
			answer, err := groqClient.AskQuestion(question, history)
			if err != nil {
				answerLabel.SetText(fmt.Sprintf("❌ Ошибка: %v", err))
				return
			}
			history = append(history, "Q: "+question, "A: "+answer)
			answerLabel.SetText(answer)
			questionEntry.SetText("")
		}()
	})

	voiceButton := widget.NewButtonWithIcon("Голос", theme.MediaRecordIcon(), func() {
		answerLabel.SetText("🎤 Запись 5 секунд...")

		go func() {
			audioFile, err := recorder.Record(5)
			if err != nil {
				answerLabel.SetText(fmt.Sprintf("❌ Ошибка записи: %v", err))
				return
			}
			defer ai.Cleanup(audioFile)

			answerLabel.SetText("📝 Распознаю...")
			text, err := groqClient.TranscribeAudio(audioFile)
			if err != nil {
				answerLabel.SetText(fmt.Sprintf("❌ Ошибка: %v", err))
				return
			}

			questionEntry.SetText(text)
			answerLabel.SetText("🎤 Распознано: " + text + "\n\n🤔 Думаю...")

			answer, err := groqClient.AskQuestion(text, history)
			if err != nil {
				answerLabel.SetText(fmt.Sprintf("❌ Ошибка: %v", err))
				return
			}

			history = append(history, "Q: "+text, "A: "+answer)
			answerLabel.SetText(answer)
		}()
	})

	clearButton := widget.NewButtonWithIcon("Очистить", theme.DeleteIcon(), func() {
		history = []string{}
		questionEntry.SetText("")
		answerLabel.SetText("Готов к работе. Задайте вопрос.")
	})

	logButton := widget.NewButtonWithIcon("Логи", theme.DocumentIcon(), func() {
		logContent, err := os.ReadFile("app.log")
		if err != nil {
			answerLabel.SetText("Лог файл не найден")
			return
		}

		logWindow := a.NewWindow("Логи приложения")
		logText := widget.NewMultiLineEntry()
		logText.SetText(string(logContent))
		logText.Wrapping = fyne.TextWrapWord

		closeBtn := widget.NewButton("Закрыть", func() {
			logWindow.Close()
		})

		logWindow.SetContent(container.NewBorder(
			nil,
			container.NewCenter(closeBtn),
			nil, nil,
			container.NewScroll(logText),
		))
		logWindow.Resize(fyne.NewSize(800, 500))
		logWindow.Show()
	})

	statusLabel := widget.NewLabel("✅ Готов к работе")
	statusLabel.Alignment = fyne.TextAlignCenter

	// Группируем кнопки в строку
	buttonRow := container.NewHBox(
		sendButton,
		voiceButton,
		clearButton,
		logButton,
	)

	// Нижняя панель: кнопки + статус
	bottomPanel := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(buttonRow),
		container.NewPadded(statusLabel),
	)

	// ========== СБОРКА ВСЕГО ИНТЕРФЕЙСА ==========
	// Используем Border для правильного распределения места:
	// - Top: поле ввода
	// - Bottom: панель кнопок
	// - Center: поле ответа (занимает всё оставшееся место)
	content := container.NewBorder(
		container.NewVBox( // Top: заголовок + поле ввода
			widget.NewLabelWithStyle("🤖 AI Interview Copilot", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabel("Вопрос:"),
			questionContainer,
		),
		bottomPanel,  // Bottom: кнопки + статус
		nil,          // Left
		nil,          // Right
		answerScroll, // Center: поле ответа (растягивается)
	)

	w.SetContent(content)
	w.ShowAndRun()
}

func showErrorWindow(msg string) {
	a := app.New()
	w := a.NewWindow("Ошибка")
	w.SetContent(container.NewCenter(widget.NewLabel(msg)))
	w.Resize(fyne.NewSize(400, 200))
	w.ShowAndRun()
}
