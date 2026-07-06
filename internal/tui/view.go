package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func composeStatus(width int, left, middle, right string) string {
	left = ansi.Strip(left)
	middle = ansi.Strip(middle)
	right = ansi.Strip(right)
	if width <= 0 {
		return ""
	}
	right = truncateCell(right, width/2)
	left = truncateCell(left, maxInt(1, width-cellWidth(right)-2))
	remaining := width - cellWidth(left) - cellWidth(right)
	if remaining <= 0 {
		return truncateCell(left+right, width)
	}
	if middle == "" {
		return left + strings.Repeat(" ", remaining) + right
	}
	middle = truncateCell(middle, remaining-2)
	line := left + " " + middle
	padding := width - cellWidth(line) - cellWidth(right)
	if padding < 1 {
		padding = 1
	}
	return truncateCell(line+strings.Repeat(" ", padding)+right, width)
}

func fitBlock(block string, width, height int) string {
	lines := strings.Split(block, "\n")
	return fitLines(lines, width, height)
}

func fitLines(lines []string, width, height int) string {
	out := make([]string, 0, height)
	for i := 0; i < height && i < len(lines); i++ {
		out = append(out, fitLine(lines[i], width))
	}
	for len(out) < height {
		out = append(out, strings.Repeat(" ", maxInt(1, width)))
	}
	return strings.Join(out, "\n")
}

func fitLine(s string, width int) string {
	s = truncateCell(s, width)
	if pad := width - cellWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func truncateCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if cellWidth(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && cellWidth(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

func cellWidth(s string) int {
	return lipgloss.Width(s)
}
