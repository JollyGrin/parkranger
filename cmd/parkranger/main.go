package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/grins/parkranger/internal/git"
	"github.com/grins/parkranger/internal/session"
	"github.com/grins/parkranger/internal/tmux"
	"github.com/grins/parkranger/internal/worktree"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	if len(args) == 0 {
		return interactive()
	}

	switch args[0] {
	case "ls", "list":
		return cmdList()
	case "open":
		if len(args) < 2 {
			return fmt.Errorf("usage: parkranger open <name>")
		}
		return cmdOpen(args[1])
	case "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: parkranger new <name>")
		}
		return cmdNew(args[1])
	case "merge":
		if len(args) < 2 {
			return fmt.Errorf("usage: parkranger merge <name>")
		}
		return cmdMerge(args[1])
	case "delete", "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: parkranger delete <name>")
		}
		return cmdDelete(args[1])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s\nRun 'parkranger help' for usage", args[0])
	}
}

func printUsage() {
	fmt.Print(`parkranger — manage parallel worktree + tmux sessions

Usage:
  parkranger              interactive picker
  parkranger ls           list worktrees with status
  parkranger open <name>  open/attach tmux session for worktree
  parkranger new <name>   create worktree + open session
  parkranger merge <name> merge worktree branch into default branch
  parkranger delete <name> kill session + remove worktree
`)
}

// resolveRepo detects the repo from CWD and returns (mainRoot, repoName, worktrees).
func resolveRepo() (mainRoot, repoName string, wts []worktree.Worktree, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", nil, fmt.Errorf("getwd: %w", err)
	}

	root, err := git.RepoRoot(cwd)
	if err != nil {
		return "", "", nil, fmt.Errorf("not in a git repo")
	}

	mainRoot, err = git.MainRepoRoot(root)
	if err != nil {
		return "", "", nil, err
	}

	repoName = git.RepoName(mainRoot)

	wts, err = worktree.List(mainRoot)
	if err != nil {
		return "", "", nil, err
	}

	return mainRoot, repoName, wts, nil
}

// --- Session picker (Bubble Tea) ---

type pickerItem struct {
	label   string           // one-line display in list
	value   string           // "live", session ID, or ""
	session *session.Session // nil for live/new — used for preview
	live    *session.LiveInfo
}

type pickerModel struct {
	title     string
	items     []pickerItem
	cursor    int
	chosen    string
	confirmed bool // true when user pressed enter (vs esc/q)
	quitting  bool
	width     int
	height    int
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.items[m.cursor].value
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	pickerTitleStyle   = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	pickerCursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	pickerDimStyle     = lipgloss.NewStyle().Faint(true)
	pickerBorderStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).PaddingLeft(1).PaddingRight(1)
	pickerHeaderStyle  = lipgloss.NewStyle().Faint(true)
)

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(pickerTitleStyle.Render(" "+m.title) + "\n")

	// List items
	for i, item := range m.items {
		if i == m.cursor {
			b.WriteString(pickerCursorStyle.Render(" > " + item.label))
		} else {
			b.WriteString(pickerDimStyle.Render("   " + item.label))
		}
		b.WriteString("\n")
	}

	// Preview box — fill remaining terminal height
	b.WriteString("\n")

	// Calculate available dimensions for the preview box
	// Title uses 2 lines (text + margin), items use len(items) lines, gap = 1
	usedLines := 2 + len(m.items) + 1
	// Border takes 2 lines (top + bottom), padding inside border ~1 line
	previewHeight := m.height - usedLines - 3
	if previewHeight < 5 {
		previewHeight = 5
	}

	// Preview width: fill terminal minus border (2) and padding (2)
	previewWidth := m.width - 4
	if previewWidth < 40 {
		previewWidth = 40
	}
	if previewWidth > 120 {
		previewWidth = 120
	}

	preview := m.renderPreview(m.items[m.cursor], previewWidth-2, previewHeight)
	style := pickerBorderStyle.Width(previewWidth)
	b.WriteString(style.Render(preview))
	b.WriteString("\n")

	return b.String()
}

func (m pickerModel) renderPreview(item pickerItem, width, maxLines int) string {
	if width < 20 {
		width = 20
	}

	// Live session
	if item.live != nil {
		status := "Session running"
		if item.live.HasClaude {
			status = fmt.Sprintf("Claude is %s", item.live.Status)
		}
		return pickerHeaderStyle.Render("● LIVE") + "\n\n" + status
	}

	// New session
	if item.session == nil {
		return pickerHeaderStyle.Render("[n] New session") + "\n\n" + "Start a fresh Claude session"
	}

	// Session preview
	s := item.session
	header := s.ID[:8]
	if s.GitBranch != "" {
		header += " · " + s.GitBranch
	}
	header += " · " + formatAge(s.ModTime)

	var body string
	if s.FullPrompt != "" {
		body = s.FullPrompt
	} else if s.FirstPrompt != "" {
		body = s.FirstPrompt
	} else {
		body = "(no prompt)"
	}

	wrapped := wordWrap(body, width)
	lines := strings.Split(wrapped, "\n")

	// Header takes 2 lines (header + blank line)
	bodyLines := maxLines - 2
	if bodyLines < 3 {
		bodyLines = 3
	}

	var content string
	if len(lines) <= bodyLines {
		// Everything fits
		content = wrapped
	} else {
		// Show beginning and end with separator
		// Give slightly more space to the beginning
		topCount := (bodyLines - 1) / 2
		bottomCount := bodyLines - 1 - topCount
		if topCount < 1 {
			topCount = 1
		}
		if bottomCount < 1 {
			bottomCount = 1
		}

		top := strings.Join(lines[:topCount], "\n")
		bottom := strings.Join(lines[len(lines)-bottomCount:], "\n")
		sep := pickerDimStyle.Render("  ···")
		content = top + "\n" + sep + "\n" + bottom
	}

	return pickerHeaderStyle.Render(header) + "\n\n" + content
}

// wordWrap breaks text at word boundaries to fit within width columns.
func wordWrap(s string, width int) string {
	var result strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		words := strings.Fields(line)
		col := 0
		for i, w := range words {
			wLen := len([]rune(w))
			if i > 0 && col+1+wLen > width {
				result.WriteString("\n")
				col = 0
			} else if i > 0 {
				result.WriteString(" ")
				col++
			}
			result.WriteString(w)
			col += wLen
		}
	}
	return result.String()
}

// sessionPicker shows a session picker for the worktree and returns a choice:
//   - "live"         → attach to existing tmux window
//   - "<session-id>" → resume with claude --resume <id>
//   - ""             → start bare claude
func sessionPicker(repoName string, wt *worktree.Worktree) (string, error) {
	sessName := tmux.SessionName(repoName)
	winName := tmux.WindowName(wt.Name)
	live := session.DetectLive(sessName, winName)

	sessions, _ := session.ListSessions(wt.Path)

	// Nothing to pick from — skip picker
	if !live.Exists && len(sessions) == 0 {
		return "", nil
	}

	// If only live session exists and no history, just attach
	if live.Exists && len(sessions) == 0 {
		return "live", nil
	}

	// Build picker items
	var items []pickerItem

	if live.Exists {
		statusStr := ""
		if live.HasClaude {
			statusStr = fmt.Sprintf(" (%s)", live.Status)
		}
		items = append(items, pickerItem{
			label: fmt.Sprintf("● LIVE%s", statusStr),
			value: "live",
			live:  &live,
		})
	}

	for i, s := range sessions {
		age := formatAge(s.ModTime)
		prompt := s.FirstPrompt
		if prompt == "" {
			prompt = s.ID[:8]
		}
		label := fmt.Sprintf("%s  %-40s  %s", s.ID[:8], prompt, age)
		items = append(items, pickerItem{
			label:   label,
			value:   s.ID,
			session: &sessions[i],
		})
	}

	items = append(items, pickerItem{
		label: "[n] New session",
		value: "",
	})

	title := fmt.Sprintf("%s / %s", repoName, wt.Name)
	model := pickerModel{title: title, items: items}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	m := result.(pickerModel)
	if !m.confirmed {
		return "", fmt.Errorf("cancelled")
	}

	return m.chosen, nil
}

// openSession creates (if needed) and attaches to a tmux window for the worktree.
func openSession(repoName, mainRoot string, wt *worktree.Worktree) error {
	sessName := tmux.SessionName(repoName)
	winName := tmux.WindowName(wt.Name)
	winTarget := tmux.WindowTarget(repoName, wt.Name)

	choice, err := sessionPicker(repoName, wt)
	if err != nil {
		return err
	}

	// Live window — just attach
	if choice == "live" {
		fmt.Printf("Attaching to %s\n", winTarget)
		return tmux.AttachWindow(sessName, winName)
	}

	// Window already exists but user picked a resume/new option
	if tmux.WindowExists(sessName, winName) {
		claudeCmd := "claude"
		if choice != "" {
			claudeCmd = fmt.Sprintf("claude --resume %s", choice)
		}
		if err := tmux.SendKeys(winTarget+".1", claudeCmd); err != nil {
			return err
		}
		return tmux.AttachWindow(sessName, winName)
	}

	// Ensure the repo-level session exists (with dashboard window 0)
	created, err := tmux.EnsureSession(repoName, mainRoot)
	if err != nil {
		return err
	}

	// Launch parkranger in the dashboard window so Ctrl-b 0 shows the TUI
	if created {
		if exe, err := os.Executable(); err == nil {
			_ = tmux.SendKeys(sessName+":dashboard", exe)
		}
	}

	// Create new window for this worktree
	fmt.Printf("Creating window %s in %s\n", winTarget, wt.Path)
	if err := tmux.CreateWindow(sessName, winName, wt.Path); err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	// Pane 0: editor with current directory
	if err := tmux.SendKeys(winTarget+".0", editor+" ."); err != nil {
		return err
	}

	// Split and launch claude in pane 1
	if err := tmux.SplitVertical(winTarget, wt.Path); err != nil {
		return err
	}

	claudeCmd := "claude"
	if choice != "" {
		claudeCmd = fmt.Sprintf("claude --resume %s", choice)
	}
	if err := tmux.SendKeys(winTarget+".1", claudeCmd); err != nil {
		return err
	}

	return tmux.AttachWindow(sessName, winName)
}

// --- Subcommands ---

func cmdList() error {
	_, repoName, wts, err := resolveRepo()
	if err != nil {
		return err
	}

	fmt.Printf(" %s\n\n", repoName)

	for _, wt := range wts {
		status := formatStatus(wt)
		sessInfo := formatSessionInfo(repoName, wt)
		marker := "  "
		if wt.IsMain {
			marker = "* "
		}

		line := fmt.Sprintf(" %s%-24s", marker, wt.Name)
		if sessInfo != "" {
			line += "  " + sessInfo
		}
		if status != "" {
			line += "  " + status
		}
		fmt.Println(line)
	}

	return nil
}

func cmdOpen(name string) error {
	mainRoot, repoName, wts, err := resolveRepo()
	if err != nil {
		return err
	}

	wt := worktree.FindByName(wts, name)
	if wt == nil {
		return fmt.Errorf("worktree %q not found", name)
	}

	return openSession(repoName, mainRoot, wt)
}

func cmdNew(name string) error {
	mainRoot, repoName, _, err := resolveRepo()
	if err != nil {
		return err
	}

	baseBranch, err := pickBaseBranch(mainRoot)
	if err != nil {
		return err
	}

	fmt.Printf("Creating worktree %q from origin/%s\n", name, baseBranch)
	wt, err := worktree.Add(mainRoot, name, "origin/"+baseBranch)
	if err != nil {
		return err
	}

	return openSession(repoName, mainRoot, &wt)
}

// pickBaseBranch shows a picker for remote branches, defaulting to the repo's default branch.
func pickBaseBranch(mainRoot string) (string, error) {
	branches, err := git.ListRemoteBranches(mainRoot)
	if err != nil || len(branches) == 0 {
		// Fallback to default branch if we can't list remotes
		b, err := git.DefaultBranch(mainRoot)
		if err != nil {
			return "", fmt.Errorf("cannot determine base branch: %w", err)
		}
		return b, nil
	}

	// Put default branch first if it's not already
	defaultBranch, _ := git.DefaultBranch(mainRoot)
	if defaultBranch != "" && len(branches) > 1 {
		for i, b := range branches {
			if b == defaultBranch && i > 0 {
				branches = append([]string{b}, append(branches[:i], branches[i+1:]...)...)
				break
			}
		}
	}

	// If only one branch, skip the picker
	if len(branches) == 1 {
		return branches[0], nil
	}

	var options []huh.Option[string]
	for _, b := range branches {
		options = append(options, huh.NewOption(b, b))
	}

	var selected string
	err = huh.NewSelect[string]().
		Title("Base branch (origin)").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return "", err
	}

	return selected, nil
}

func cmdMerge(name string) error {
	mainRoot, _, wts, err := resolveRepo()
	if err != nil {
		return err
	}

	wt := worktree.FindByName(wts, name)
	if wt == nil {
		return fmt.Errorf("worktree %q not found", name)
	}
	if wt.IsMain {
		return fmt.Errorf("cannot merge the main worktree")
	}

	defaultBranch, err := git.DefaultBranch(mainRoot)
	if err != nil {
		return err
	}

	var confirm bool
	err = huh.NewConfirm().
		Title(fmt.Sprintf("Merge %s into %s?", wt.Branch, defaultBranch)).
		Value(&confirm).
		Run()
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	fmt.Printf("Merging %s into %s\n", wt.Branch, defaultBranch)
	if err := git.MergeBranch(mainRoot, wt.Branch, defaultBranch); err != nil {
		return err
	}

	// Offer to clean up
	var cleanup bool
	err = huh.NewConfirm().
		Title("Delete worktree and branch?").
		Value(&cleanup).
		Run()
	if err != nil {
		return err
	}
	if !cleanup {
		fmt.Println("Merge complete. Worktree kept.")
		return nil
	}

	sessName := tmux.SessionName(git.RepoName(mainRoot))
	winName := tmux.WindowName(wt.Name)
	if tmux.WindowExists(sessName, winName) {
		tmux.KillWindow(sessName, winName)
	}

	if err := worktree.Remove(mainRoot, wt.Path); err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}

	if err := git.DeleteBranch(mainRoot, wt.Branch); err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}

	fmt.Println("Merge complete. Worktree and branch deleted.")
	return nil
}

func cmdDelete(name string) error {
	mainRoot, _, wts, err := resolveRepo()
	if err != nil {
		return err
	}

	wt := worktree.FindByName(wts, name)
	if wt == nil {
		return fmt.Errorf("worktree %q not found", name)
	}
	if wt.IsMain {
		return fmt.Errorf("cannot delete the main worktree")
	}

	var confirm bool
	err = huh.NewConfirm().
		Title(fmt.Sprintf("Delete worktree %q and kill tmux session?", name)).
		Value(&confirm).
		Run()
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	sessName := tmux.SessionName(git.RepoName(mainRoot))
	winName := tmux.WindowName(wt.Name)
	if tmux.WindowExists(sessName, winName) {
		fmt.Printf("Killing window %s:%s\n", sessName, winName)
		tmux.KillWindow(sessName, winName)
	}

	fmt.Printf("Removing worktree %s\n", wt.Path)
	if err := worktree.Remove(mainRoot, wt.Path); err != nil {
		return err
	}

	if err := git.DeleteBranch(mainRoot, wt.Branch); err != nil {
		fmt.Printf("Warning: could not delete branch %s: %v\n", wt.Branch, err)
	} else {
		fmt.Printf("Deleted branch %s\n", wt.Branch)
	}

	return nil
}

// --- Interactive mode ---

type menuChoice struct {
	action string // "open", "new", "merge", "delete"
	name   string // worktree name (for open)
}

type menuItem struct {
	name    string
	live    session.LiveInfo
	sessNum int
	ahead   int
	behind  int
	dirty   bool
	isMain  bool
	choice  menuChoice
}

type menuModel struct {
	title     string
	items     []menuItem
	cursor    int
	selected  menuChoice
	confirmed bool
	quitting  bool
	width     int
	height    int
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.items[m.cursor].choice
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "n":
			m.selected = menuChoice{action: "new"}
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "m":
			m.selected = menuChoice{action: "merge"}
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "d":
			m.selected = menuChoice{action: "delete"}
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	menuPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)

	menuTitleStyle = lipgloss.NewStyle().Bold(true)
	menuDimStyle   = lipgloss.NewStyle().Faint(true)

	menuAccentColor  = lipgloss.Color("4")
	menuIdleColor    = lipgloss.Color("2")
	menuBusyColor    = lipgloss.Color("4")
	menuWaitingColor = lipgloss.Color("3")
)

func statusColor(s session.AgentStatus) lipgloss.Color {
	switch s {
	case session.StatusIdle:
		return menuIdleColor
	case session.StatusBusy:
		return menuBusyColor
	case session.StatusWaiting:
		return menuWaitingColor
	default:
		return lipgloss.Color("8")
	}
}

func (m menuModel) View() string {
	if m.quitting {
		return ""
	}

	// Column widths from data
	maxName := 0
	for _, item := range m.items {
		if n := len(item.name); n > maxName {
			maxName = n
		}
	}
	if maxName < 12 {
		maxName = 12
	}

	// Build rows
	var rows []string
	for i, item := range m.items {
		// Cursor
		var cursor string
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(menuAccentColor).Render("▸ ")
		} else {
			cursor = "  "
		}

		// Name
		padded := fmt.Sprintf("%-*s", maxName, item.name)
		var name string
		if i == m.cursor {
			name = lipgloss.NewStyle().Bold(true).Render(padded)
		} else {
			name = menuDimStyle.Render(padded)
		}

		// Status indicator
		statusWidth := 10
		var statusCol string
		if item.live.HasClaude {
			c := statusColor(item.live.Status)
			dot := lipgloss.NewStyle().Foreground(c).Render("●")
			label := item.live.Status.String()
			statusCol = dot + " " + lipgloss.NewStyle().Foreground(c).Render(label)
			if pad := statusWidth - 2 - len(label); pad > 0 {
				statusCol += strings.Repeat(" ", pad)
			}
		} else if item.live.Exists {
			statusCol = menuDimStyle.Render("● live")
			statusCol += strings.Repeat(" ", statusWidth-6)
		} else {
			statusCol = strings.Repeat(" ", statusWidth)
		}

		// Session count
		sessWidth := 12
		var sessCol string
		if item.sessNum > 0 {
			noun := "sessions"
			if item.sessNum == 1 {
				noun = "session"
			}
			s := fmt.Sprintf("%d %s", item.sessNum, noun)
			sessCol = menuDimStyle.Render(s)
			if pad := sessWidth - len(s); pad > 0 {
				sessCol += strings.Repeat(" ", pad)
			}
		} else {
			sessCol = strings.Repeat(" ", sessWidth)
		}

		// Git badges
		var badges []string
		if item.ahead > 0 {
			badges = append(badges, menuDimStyle.Render(fmt.Sprintf("↑%d", item.ahead)))
		}
		if item.behind > 0 {
			badges = append(badges, menuDimStyle.Render(fmt.Sprintf("↓%d", item.behind)))
		}
		if item.dirty {
			badges = append(badges, lipgloss.NewStyle().Foreground(menuWaitingColor).Render("✱"))
		}
		gitCol := strings.Join(badges, " ")

		row := cursor + name + "  " + statusCol + "  " + sessCol + gitCol
		rows = append(rows, row)
	}

	// Title
	title := menuTitleStyle.Render("parkranger") +
		menuDimStyle.Render(" · "+m.title)

	// Keybind hints with accented keys
	accent := lipgloss.NewStyle().Foreground(menuAccentColor)
	hints := accent.Render("n") + menuDimStyle.Render(" new") + "   " +
		accent.Render("m") + menuDimStyle.Render(" merge") + "   " +
		accent.Render("d") + menuDimStyle.Render(" delete") + "   " +
		accent.Render("q") + menuDimStyle.Render(" quit")

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" + hints
	panel := menuPanelStyle.Render(content)

	// Center in terminal
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			panel)
	}

	return "\n" + panel + "\n"
}

func interactive() error {
	for {
		_, repoName, wts, err := resolveRepo()
		if err != nil {
			return err
		}

		var items []menuItem
		for _, wt := range wts {
			sessName := tmux.SessionName(repoName)
			winName := tmux.WindowName(wt.Name)
			live := session.DetectLive(sessName, winName)
			sessions, _ := session.ListSessions(wt.Path)

			items = append(items, menuItem{
				name:    wt.Name,
				live:    live,
				sessNum: len(sessions),
				ahead:   wt.Ahead,
				behind:  wt.Behind,
				dirty:   wt.Dirty,
				isMain:  wt.IsMain,
				choice:  menuChoice{action: "open", name: wt.Name},
			})
		}

		model := menuModel{title: repoName, items: items}
		p := tea.NewProgram(model, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			return err
		}

		m := result.(menuModel)
		if !m.confirmed {
			return nil
		}

		switch m.selected.action {
		case "open":
			// open attaches to tmux — if outside tmux, syscall.Exec replaces
			// the process so the loop won't continue (which is fine).
			// If inside tmux, switch-client returns and we loop back.
			if err := cmdOpen(m.selected.name); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "new":
			var name string
			err := huh.NewInput().
				Title("Branch name").
				Value(&name).
				Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				continue
			}
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if err := cmdNew(name); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "merge":
			name, err := pickWorktree(wts, "Merge which worktree?")
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				continue
			}
			if name == "" {
				continue
			}
			if err := cmdMerge(name); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "delete":
			name, err := pickWorktree(wts, "Delete which worktree?")
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				continue
			}
			if name == "" {
				continue
			}
			if err := cmdDelete(name); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		}
	}
}

func pickWorktree(wts []worktree.Worktree, title string) (string, error) {
	var options []huh.Option[string]
	for _, wt := range wts {
		if wt.IsMain {
			continue
		}
		label := fmt.Sprintf("%-24s %s", wt.Name, formatStatus(wt))
		options = append(options, huh.NewOption(label, wt.Name))
	}

	if len(options) == 0 {
		fmt.Println("No worktrees to select (only main).")
		return "", nil
	}

	var name string
	err := huh.NewSelect[string]().
		Title(title).
		Options(options...).
		Value(&name).
		Run()
	return name, err
}

// formatSessionInfo returns a string like "● 1 live, 7 sessions" or "3 sessions".
func formatSessionInfo(repoName string, wt worktree.Worktree) string {
	sessName := tmux.SessionName(repoName)
	winName := tmux.WindowName(wt.Name)
	live := session.DetectLive(sessName, winName)

	sessions, _ := session.ListSessions(wt.Path)
	count := len(sessions)

	var parts []string

	if live.HasClaude {
		parts = append(parts, fmt.Sprintf("● %s", live.Status))
	} else if live.Exists {
		parts = append(parts, "● live")
	}

	if count > 0 {
		noun := "sessions"
		if count == 1 {
			noun = "session"
		}
		parts = append(parts, fmt.Sprintf("%d %s", count, noun))
	}

	return strings.Join(parts, ", ")
}

func formatStatus(wt worktree.Worktree) string {
	var parts []string

	if wt.Branch != "" && wt.Branch != wt.Name {
		parts = append(parts, wt.Branch)
	}

	if wt.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%d ahead", wt.Ahead))
	}
	if wt.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%d behind", wt.Behind))
	}
	if wt.Dirty {
		parts = append(parts, "dirty")
	}

	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// formatAge converts a time to a human-readable age string.
func formatAge(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
