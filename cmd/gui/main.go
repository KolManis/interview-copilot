// cmd/gui/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"

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
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Не удалось создать файл логов:", err)
	} else {
		log.SetOutput(logFile)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// ========== ФУНКЦИИ ДЛЯ STEALTH MODE ==========

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	findWindowW              = user32.NewProc("FindWindowW")
	setWindowDisplayAffinity = user32.NewProc("SetWindowDisplayAffinity")
)

const WDA_EXCLUDEFROMCAPTURE = 0x00000011

// MakeWindowStealth делает окно невидимым для демонстрации экрана
func MakeWindowStealth(hwnd uintptr) {
	// Исключаем из захвата экрана (интервьюер НЕ ВИДИТ окно на screen share)
	setWindowDisplayAffinity.Call(hwnd, uintptr(WDA_EXCLUDEFROMCAPTURE))
	log.Println("✅ Stealth mode activated - окно скрыто от демонстрации экрана")
}

// FindWindowByTitle ищет окно по заголовку
func FindWindowByTitle(title string) uintptr {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	hwnd, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	return hwnd
}

// Функция для форматирования ответа AI в удобочитаемый вид
func formatAnswerForFyne(text string) string {
	// 1. Обрабатываем блоки кода ```language ... ```
	codeBlockRegex := regexp.MustCompile("(?s)```(?:\\w+)?\\n(.*?)```")
	text = codeBlockRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Извлекаем код без маркеров
		codeMatch := regexp.MustCompile("```(?:\\w+)?\\n(.*?)```").FindStringSubmatch(match)
		if len(codeMatch) > 1 {
			code := codeMatch[1]
			return "\n╔════════════════════════════════════════════════════╗\n║ КОД:                                               ║\n╠════════════════════════════════════════════════════╣\n" +
				indentLines(code, "║ ") + "\n" +
				"╚════════════════════════════════════════════════════╝\n"
		}
		return match
	})

	// 2. Обрабатываем строчные блоки кода `code`
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")
	text = inlineCodeRegex.ReplaceAllString(text, "「$1」")

	// 3. Жирный текст **text** → заменяем на заглавные/маркеры
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	text = boldRegex.ReplaceAllString(text, "▸ $1 ◂")

	// 4. Курсив *text* → выделяем стрелочками
	italicRegex := regexp.MustCompile(`\*([^*]+)\*`)
	text = italicRegex.ReplaceAllString(text, "‹ $1 ›")

	// 5. Списки
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Обработка маркеров списков
		if strings.HasPrefix(trimmed, "- ") {
			line = "  • " + strings.TrimPrefix(trimmed, "- ")
		} else if strings.HasPrefix(trimmed, "• ") {
			line = "  • " + strings.TrimPrefix(trimmed, "• ")
		} else if strings.HasPrefix(trimmed, "* ") && !strings.HasPrefix(trimmed, "**") {
			line = "  • " + strings.TrimPrefix(trimmed, "* ")
		} else if strings.HasPrefix(trimmed, "1.") || strings.HasPrefix(trimmed, "2.") ||
			strings.HasPrefix(trimmed, "3.") || strings.HasPrefix(trimmed, "4.") {
			line = "  " + trimmed
		}

		result = append(result, line)
	}
	text = strings.Join(result, "\n")

	// 6. Горизонтальные разделители
	text = strings.ReplaceAll(text, "---", "\n"+strings.Repeat("─", 60)+"\n")

	// 7. Убираем множественные пустые строки
	text = regexp.MustCompile("\n{3,}").ReplaceAllString(text, "\n\n")

	// 8. Обрезаем слишком длинные строки (Fyne не любит очень длинные строки)
	lines2 := strings.Split(text, "\n")
	for i, line := range lines2 {
		if len(line) > 120 {
			// Разбиваем длинные строки
			runes := []rune(line)
			var newLines []string
			for j := 0; j < len(runes); j += 100 {
				end := j + 100
				if end > len(runes) {
					end = len(runes)
				}
				newLines = append(newLines, string(runes[j:end]))
			}
			lines2[i] = strings.Join(newLines, "\n")
		}
	}
	text = strings.Join(lines2, "\n")

	return text
}

// Вспомогательная функция для отступа строк
func indentLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
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
	w.Resize(fyne.NewSize(1000, 800))

	// ========== ПОЛЕ ВВОДА ==========
	questionEntry := widget.NewMultiLineEntry()
	questionEntry.SetPlaceHolder("Введите ваш вопрос здесь...")
	questionEntry.Wrapping = fyne.TextWrapWord
	questionContainer := container.NewPadded(questionEntry)

	// ========== ПОЛЕ ОТВЕТА ==========
	answerLabel := widget.NewLabel("")
	answerLabel.Wrapping = fyne.TextWrapWord
	answerScroll := container.NewScroll(answerLabel)
	answerScroll.SetMinSize(fyne.NewSize(950, 500))

	// Функция установки ответа
	setAnswer := func(text string) {
		formatted := formatAnswerForFyne(text)
		log.Printf("Длина ответа: %d символов", len(formatted))
		answerLabel.SetText(formatted)
		answerScroll.ScrollToTop()
	}

	// ========== ИСТОРИЯ ==========
	var history []string

	// ========== КНОПКИ ==========
	sendButton := widget.NewButtonWithIcon("Отправить", theme.NavigateNextIcon(), func() {
		question := strings.TrimSpace(questionEntry.Text)
		if question == "" {
			return
		}
		setAnswer("🤔 Думаю...")
		go func() {
			answer, err := groqClient.AskQuestion(question, history)
			if err != nil {
				setAnswer(fmt.Sprintf("❌ Ошибка: %v", err))
				return
			}
			history = append(history, "Q: "+question, "A: "+answer)
			setAnswer(answer)
			questionEntry.SetText("")
		}()
	})

	voiceButton := widget.NewButtonWithIcon("Голос", theme.MediaRecordIcon(), func() {
		setAnswer("🎤 Запись 5 секунд...")
		go func() {
			audioFile, err := recorder.Record(5)
			if err != nil {
				setAnswer(fmt.Sprintf("❌ Ошибка записи: %v", err))
				return
			}
			defer ai.Cleanup(audioFile)
			setAnswer("📝 Распознаю речь...")
			text, err := groqClient.TranscribeAudio(audioFile)
			if err != nil {
				setAnswer(fmt.Sprintf("❌ Ошибка распознавания: %v", err))
				return
			}
			questionEntry.SetText(text)
			setAnswer(fmt.Sprintf("🎤 Распознано: %s\n\n🤔 Думаю...", text))
			answer, err := groqClient.AskQuestion(text, history)
			if err != nil {
				setAnswer(fmt.Sprintf("❌ Ошибка: %v", err))
				return
			}
			history = append(history, "Q: "+text, "A: "+answer)
			setAnswer(answer)
		}()
	})

	clearButton := widget.NewButtonWithIcon("Очистить", theme.DeleteIcon(), func() {
		history = []string{}
		questionEntry.SetText("")
		setAnswer("✅ История очищена. Задайте новый вопрос.")
	})

	logButton := widget.NewButtonWithIcon("Логи", theme.DocumentIcon(), func() {
		logContent, err := os.ReadFile("app.log")
		if err != nil {
			setAnswer("Лог файл не найден")
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

	statusLabel := widget.NewLabel("✅ Готов к работе (Stealth Mode)")
	statusLabel.Alignment = fyne.TextAlignCenter

	buttonRow := container.NewHBox(sendButton, voiceButton, clearButton, logButton)
	bottomPanel := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(buttonRow),
		container.NewPadded(statusLabel),
	)

	// ========== СБОРКА ==========
	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("🤖 AI Interview Copilot", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Вопрос:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			questionContainer,
		),
		bottomPanel,
		nil,
		nil,
		answerScroll,
	)

	w.SetContent(content)

	// ========== STEALTH MODE ==========
	w.Show() // Показываем окно

	// Включаем невидимость после создания окна
	go func() {
		time.Sleep(500 * time.Millisecond)
		hwnd := FindWindowByTitle("AI Interview Copilot")
		if hwnd != 0 {
			MakeWindowStealth(hwnd)
			log.Println("🔒 Stealth mode активен - окно скрыто от screen share")
		} else {
			log.Println("⚠️ Stealth mode не активирован - окно не найдено")
		}
	}()

	w.ShowAndRun()
}

func showErrorWindow(msg string) {
	a := app.New()
	w := a.NewWindow("Ошибка")
	w.SetContent(container.NewCenter(widget.NewLabel(msg)))
	w.Resize(fyne.NewSize(400, 200))
	w.ShowAndRun()
}
