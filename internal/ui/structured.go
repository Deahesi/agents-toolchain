package ui

import (
	"strings"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Field is one labeled, preformatted value. Its order in the slice is preserved.
type Field = domain.Field

// PrintFields prints an optional title followed by aligned labels and values.
// Multiline values remain in the value column. Empty input produces no output.
func (s *UIService) PrintFields(title string, fields []Field) {
	renderer := lipgloss.NewRenderer(s.output)
	var sections []string
	if title != "" {
		sections = append(sections, renderer.NewStyle().Bold(true).Render(title))
	}
	if len(fields) > 0 {
		rows := make([][]string, len(fields))
		for i, field := range fields {
			rows[i] = []string{field.Label, field.Value}
		}
		labels := renderer.NewStyle().Foreground(lipgloss.Color("8"))
		values := renderer.NewStyle()
		text := renderColumns(rows, func(_, column int) lipgloss.Style {
			if column == 0 {
				return labels
			}
			return values
		})
		sections = append(sections, renderer.NewStyle().PaddingLeft(2).Render(text))
	}
	s.printStructured(strings.Join(sections, "\n"))
}

// PrintTable prints aligned columns, with optional headers. Missing cells are
// empty; extra cells are retained under blank headers. Multiline cells and their
// order are preserved. The caller's headers and rows are never modified.
func (s *UIService) PrintTable(headers []string, rows [][]string) {
	data := make([][]string, 0, len(rows)+1)
	if len(headers) > 0 {
		data = append(data, headers)
	}
	data = append(data, rows...)
	renderer := lipgloss.NewRenderer(s.output)
	text := renderColumns(data, func(row, _ int) lipgloss.Style {
		style := renderer.NewStyle()
		if len(headers) > 0 && row == 0 {
			style = style.Bold(true).Foreground(lipgloss.Color("6"))
		}
		return style
	})
	s.printStructured(text)
}

// Use the existing log path so a complete block is flushed without disrupting
// live spinners and response areas. Redirected output contains plain text.
func (s *UIService) printStructured(text string) {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return
	}
	if !s.interactive {
		text = ansi.Strip(text)
	}
	s.log(s.output, text)
}

// Size by terminal cell width, so wide Unicode and ANSI styles do not shift
// subsequent columns. JoinHorizontal aligns multiline cells at the top.
func renderColumns(rows [][]string, style func(row, column int) lipgloss.Style) string {
	var widths []int
	data := make([][]string, len(rows))
	for row, cells := range rows {
		data[row] = make([]string, len(cells))
		for column, cell := range cells {
			cell = strings.ReplaceAll(strings.ReplaceAll(cell, "\r\n", "\n"), "\t", "    ")
			data[row][column] = cell
			if column == len(widths) {
				widths = append(widths, 0)
			}
			widths[column] = max(widths[column], lipgloss.Width(cell))
		}
	}
	if len(widths) == 0 {
		return ""
	}
	lines := make([]string, len(data))
	for row, cells := range data {
		var columns []string
		for column, width := range widths {
			if column > 0 {
				columns = append(columns, "  ")
			}
			var cell string
			if column < len(cells) {
				cell = cells[column]
			}
			columns = append(columns, style(row, column).Width(width).Render(cell))
		}
		lines[row] = lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	}
	return strings.Join(lines, "\n")
}
