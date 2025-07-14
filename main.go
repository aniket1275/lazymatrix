package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type focusedList int

const (
	leftList focusedList = iota
	centerList
	rightList
)

type model struct {
	listA, listB, listC list.Model
	focus               focusedList
	width, height       int
	store               *Store
	textView            bool
	textInputs          []textinput.Model
	inputIndex          int
	selectedItem        list.Item
	editMode            bool
	editItem            item
	editSection         string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) isFiltering() bool {
	return m.listA.FilterState() == list.Filtering ||
		m.listB.FilterState() == list.Filtering ||
		m.listC.FilterState() == list.Filtering
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.textView {
		// Form input mode
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if m.inputIndex == len(m.textInputs)-1 {
					allFilled := true
					for _, input := range m.textInputs {
						if input.Value() == "" {
							allFilled = false
							break
						}
					}

					if allFilled {
						title := m.textInputs[0].Value()
						desc := m.textInputs[1].Value()
						section := m.textInputs[2].Value()
						if m.editMode {
							err := m.store.Update(title, desc, section, m.editItem.title)
							if err != nil {
								log.Println("Update failed:", err)
							}
							m.editMode = false
						} else {
							err := m.store.Save(title, desc, section)
							if err != nil {
								log.Println("Insert failed:", err)
							}
						}

						for i := range m.textInputs {
							m.textInputs[i].SetValue("")
							m.textInputs[i].Blur()
						}
						m.inputIndex = 0

						m.textView = false

						m.listA.SetItems(FetchItemsBySection(m.store.conn, "A"))
						m.listB.SetItems(FetchItemsBySection(m.store.conn, "B"))
						m.listC.SetItems(FetchItemsBySection(m.store.conn, "C"))

					}
				} else {
					m.inputIndex++
					for i := range m.textInputs {
						if i == m.inputIndex {
							m.textInputs[i].Focus()
						} else {
							m.textInputs[i].Blur()
						}
					}
				}

			case "esc":
				m.textView = false
			}
		}

		// Update all text inputs
		for i := range m.textInputs {
			m.textInputs[i], _ = m.textInputs[i].Update(msg)
		}
		return m, nil
	}

	// Normal list navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "d":
			if !m.isFiltering() {
				var selectedItem list.Item
				var section string

				switch m.focus {
				case leftList:
					selectedItem = m.listA.SelectedItem()
					section = "A"
				case centerList:
					selectedItem = m.listC.SelectedItem()
					section = "C"
				case rightList:
					selectedItem = m.listB.SelectedItem()
					section = "B"
				}

				if selectedItem != nil {
					title := selectedItem.FilterValue()

					err := m.store.Delete(title, section)
					if err != nil {
						log.Println("Delete failed:", err)
					}

					// Reload the list after deletion
					m.listA.SetItems(FetchItemsBySection(m.store.conn, "A"))
					m.listB.SetItems(FetchItemsBySection(m.store.conn, "B"))
					m.listC.SetItems(FetchItemsBySection(m.store.conn, "C"))
				}
			}
		case "ctrl+c", "q":
			if !m.isFiltering() {
				return m, tea.Quit
			}
		case "h":
			if !m.isFiltering() {
				m.focus = (m.focus + 1) % 3
			}
		case "l":
			if !m.isFiltering() {
				m.focus = (m.focus + 2) % 3
			}
		case "a":
			if !m.isFiltering() {
				m.textView = true
				m.inputIndex = 0
				for i := range m.textInputs {
					if i == 0 {
						m.textInputs[i].Focus()
					} else {
						m.textInputs[i].Blur()
					}
				}
			}
		case "u":
			if !m.isFiltering() {
				var selectedItem list.Item
				var section string

				switch m.focus {
				case leftList:
					selectedItem = m.listA.SelectedItem()
					section = "A"
				case centerList:
					selectedItem = m.listC.SelectedItem()
					section = "C"
				case rightList:
					selectedItem = m.listB.SelectedItem()
					section = "B"
				}

				if selectedItem != nil {
					i := selectedItem.(item)
					m.editItem = i
					m.editSection = section
					m.editMode = true
					m.textView = true

					m.textInputs[0].SetValue(i.title)
					m.textInputs[1].SetValue(i.desc)
					m.textInputs[2].SetValue(section)

					m.inputIndex = 0
					for i := range m.textInputs {
						if i == 0 {
							m.textInputs[i].Focus()
						} else {
							m.textInputs[i].Blur()
						}
					}
				}

			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		listWidth := int(0.3 * float64(msg.Width))
		listHeight := int(0.75 * float64(msg.Height))

		frame := focusedStyle.GetHorizontalFrameSize()
		contentWidth := listWidth - frame

		m.listA.SetSize(contentWidth, listHeight)
		m.listB.SetSize(contentWidth, listHeight)
		m.listC.SetSize(contentWidth, listHeight)
	}

	switch m.focus {
	case leftList:
		m.listA, cmd = m.listA.Update(msg)
	case centerList:
		m.listC, cmd = m.listC.Update(msg)
	case rightList:
		m.listB, cmd = m.listB.Update(msg)
	}

	return m, cmd
}

var (
	unfocusedStyle = lipgloss.NewStyle().
			Padding(2).
			Margin(2)

	focusedStyle = lipgloss.NewStyle().
			Margin(2).
			Padding(2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))
)

func (m model) View() string {
	left := unfocusedStyle.Width(int(0.3 * float64(m.width))).Render(m.listA.View())
	center := unfocusedStyle.Width(int(0.3 * float64(m.width))).Render(m.listC.View())
	right := unfocusedStyle.Width(int(0.3 * float64(m.width))).Render(m.listB.View())

	if m.textView {
		var inputViews []string
		for _, ti := range m.textInputs {
			inputViews = append(inputViews, ti.View())
		}

		form := lipgloss.JoinVertical(lipgloss.Left, inputViews...)
		box := lipgloss.NewStyle().
			Width(100).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(form + "\n\nPress Enter to continue or Esc to cancel")

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	switch m.focus {
	case leftList:
		left = focusedStyle.Width(int(0.3 * float64(m.width))).Render(m.listA.View())
	case centerList:
		center = focusedStyle.Width(int(0.3 * float64(m.width))).Render(m.listC.View())
	case rightList:
		right = focusedStyle.Width(int(0.3 * float64(m.width))).Render(m.listB.View())
	}

	listsRow := lipgloss.JoinHorizontal(lipgloss.Top, left, right, center)
	listsCentered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, listsRow)

	helpText := "  /: search • l: forward • h: backward • a: add • d: delete • u: update • q: quit "
	helpStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(helpText)
	help := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, helpStyled)

	return lipgloss.JoinVertical(lipgloss.Top, listsCentered, help)
}

func main() {
	// Items for each list
	store := &Store{}
	if err := store.Init(); err != nil {
		log.Fatalf("unable to run tui %v", err)
	}
	// store.Seed()

	itemsA := FetchItemsBySection(store.conn, "A")
	itemsB := FetchItemsBySection(store.conn, "B")
	itemsC := FetchItemsBySection(store.conn, "C")
	// Create lists and remove help
	listA := list.New(itemsA, list.NewDefaultDelegate(), 0, 0)
	listA.Title = "Do Now"
	listA.SetShowHelp(false)

	listB := list.New(itemsB, list.NewDefaultDelegate(), 0, 0)
	listB.Title = "Schedule"
	listB.SetShowHelp(false)

	listC := list.New(itemsC, list.NewDefaultDelegate(), 0, 0)
	listC.Title = "Delegate"
	listC.SetShowHelp(false)

	m := model{
		listA:    listA,
		listB:    listB,
		listC:    listC,
		focus:    leftList,
		store:    store,
		textView: false,
	}
	inputs := make([]textinput.Model, 3)
	labels := []string{"Title", "Description", "Section"}

	for i := range inputs {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.Prompt = labels[i] + ": "
		ti.CharLimit = 64
		ti.Width = 40

		if i == 0 {
			ti.Focus()
			ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
			ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
		}

		inputs[i] = ti
	}

	m.textInputs = inputs
	m.inputIndex = 0

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
