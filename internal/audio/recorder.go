package audio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Recorder struct {
	TempDir string
}

func NewRecorder() *Recorder {
	return &Recorder{
		TempDir: os.TempDir(),
	}
}

func (r *Recorder) Record(duration int) (string, error) {
	outputFile := filepath.Join(r.TempDir, "copilot_audio.wav")

	fmt.Printf("🎤 Записываю %d сек... (говори вопрос)\n", duration)

	// Твой микрофон из списка устройств
	micName := "Набор микрофонов (Realtek(R) Audio)"

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-f", "dshow",
		"-i", fmt.Sprintf("audio=%s", micName),
		"-t", fmt.Sprintf("%d", duration),
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ошибка записи: %v\nВывод ffmpeg: %s", err, string(output))
	}

	fmt.Println("✅ Запись завершена")
	return outputFile, nil
}
