// cmd/gui/styles.go
package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type MonochromeTheme struct{}

func (m MonochromeTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch c {
	case theme.ColorNameBackground:
		return color.RGBA{0, 0, 0, 255} // Черный фон
	case theme.ColorNameForeground:
		return color.RGBA{255, 255, 255, 255} // Белый текст
	case theme.ColorNameButton:
		return color.RGBA{50, 50, 50, 255} // Темно-серые кнопки
	case theme.ColorNamePrimary:
		return color.RGBA{200, 200, 200, 255} // Светло-серый акцент
	default:
		return theme.DefaultTheme().Color(c, v)
	}
}

func (m MonochromeTheme) Size(s fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(s)
}

func (m MonochromeTheme) Font(s fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(s)
}

// Применение темы:
// a.Settings().SetTheme(MonochromeTheme{})
