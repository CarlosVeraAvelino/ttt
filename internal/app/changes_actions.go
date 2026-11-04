package app

import (
	"fmt"
	"strings"

	"github.com/eugenioenko/ttt/internal/command"
	"github.com/eugenioenko/ttt/internal/git"
	"github.com/eugenioenko/ttt/internal/ui"
	"github.com/eugenioenko/ttt/internal/widgets"
)

// showNamedMenu opens a context menu of names under (x, y). The entry for
// current is checked, and picking a name calls onPick with it. Each entry gets
// a command ID under prefix, and reopening the menu replaces those IDs.
func (a *App) showNamedMenu(prefix string, names []string, current string, x, y int, onPick func(string)) {
	items := make([]ui.ContextMenuItem, 0, len(names))
	for i, name := range names {
		id := fmt.Sprintf("%s.%d", prefix, i)
		picked := name
		a.Reg.Register(command.Command{
			ID: id, Title: picked,
			Handler: func() { onPick(picked) },
		})
		checked := ui.MenuUnchecked
		if name == current {
			checked = ui.MenuChecked
		}
		items = append(items, ui.ContextMenuItem{Label: name, Command: id, Checked: checked})
	}
	openContextMenu(a, items, x, y)
}

// ShowBranchMenu lists the local branches of dir, the checked-out one ticked,
// and ends with New Branch. Picking a branch switches to it.
func (a *App) ShowBranchMenu(dir string, x, y int) {
	branches, err := git.LocalBranches(dir)
	if err != nil {
		a.StatusError("Could not list branches: " + err.Error())
		return
	}
	names := make([]string, 0, len(branches))
	current := ""
	for _, b := range branches {
		names = append(names, b.Name)
		if b.Current {
			current = b.Name
		}
	}
	a.showBranchItems(dir, names, current, x, y)
}

// showBranchItems is the branch menu: a row per branch, the current one
// ticked, then New Branch, which runs the existing git.newBranch command.
func (a *App) showBranchItems(dir string, names []string, current string, x, y int) {
	items := make([]ui.ContextMenuItem, 0, len(names)+2)
	for i, name := range names {
		id := fmt.Sprintf("changes.branch.%d", i)
		picked := name
		a.Reg.Register(command.Command{
			ID: id, Title: "Switch to " + picked,
			Handler: func() {
				if picked == current {
					return
				}
				a.RunRepoTask(RepoTask{
					Progress: "Switching branch", Done: "Switched to " + picked,
					Dirs: []string{dir}, Ops: []RepoOp{OpSwitchBranch(picked)},
				})
			},
		})
		checked := ui.MenuUnchecked
		if name == current {
			checked = ui.MenuChecked
		}
		items = append(items, ui.ContextMenuItem{Label: name, Command: id, Checked: checked})
	}
	newID := "changes.branch.new"
	a.Reg.Register(command.Command{
		ID: newID, Title: "New Branch",
		Handler: func() { a.ShowNewBranchDialog(dir) },
	})
	items = append(items, ui.MenuSep(), ui.ContextMenuItem{Label: "New Branch…", Command: newID})
	openContextMenu(a, items, x, y)
}

// ShowNewBranchDialog asks for a name and the branch to start it from. The
// starting point is the checked-out branch unless a menu picks another local
// or origin branch.
func (a *App) ShowNewBranchDialog(dir string) {
	local, err := git.LocalBranches(dir)
	if err != nil {
		a.StatusError("Could not list branches: " + err.Error())
		return
	}
	remote, _ := git.RemoteBranches(dir)
	options := make([]string, 0, len(local)+len(remote))
	base := ""
	for _, b := range local {
		options = append(options, b.Name)
		if b.Current {
			base = b.Name
		}
	}
	if base == "" && len(options) > 0 {
		base = options[0]
	}
	for _, name := range remote {
		options = append(options, "origin/"+name)
	}

	name := widgets.NewInputWidget(widgets.InputConfig{Placeholder: "Branch name"})
	var baseButton *widgets.ButtonWidget
	baseButton = menuButtonAt("From: "+base, func(x, y int) {
		a.showNamedMenu("changes.newbranch.base", options, base, x, y, func(picked string) {
			base = picked
			baseButton.SetLabel("From: " + picked)
		})
	})

	dialog := widgets.NewDialogWidget(60)
	dialog.Title = "New Branch"
	dialog.Borders = *a.Borders
	dialog.SetContent(widgets.NewVStackWidget(name, baseButton))
	dialog.Buttons = []widgets.DialogButton{
		{Label: "&Cancel", Handler: func() { a.DismissDialog() }},
		{Label: "&Create", Handler: func() {
			branch := strings.TrimSpace(name.Text())
			if branch == "" {
				a.StatusWarn("A branch needs a name")
				return
			}
			a.DismissDialog()
			a.RunRepoTask(RepoTask{
				Progress: "Creating branch", Done: "Created branch: " + branch,
				Dirs: []string{dir}, Ops: []RepoOp{OpCreateBranchFrom(branch, base)},
			})
		}},
	}
	dialog.OnDismiss = func() { a.DismissDialog() }
	dialog.Build()

	a.ShowDialog(ui.NewWidgetAdapter(dialog))
}

// ShowChangesSidebar shows the sidebar's Changes panel, as its activity bar
// entry does. The repository's changes load on their own.
func (a *App) ShowChangesSidebar() {
	a.ShowPanel("changes", a.Changes.Adapter)
}

// ShowPullRequestDialog asks for a base branch on origin, the branch to
// compare, a title and a description, then opens the pull request with gh.
func (a *App) ShowPullRequestDialog(dir string) {
	remote, err := git.RemoteBranches(dir)
	if err != nil || len(remote) == 0 {
		a.StatusWarn("No branches on origin to open a pull request against")
		return
	}
	local, err := git.LocalBranches(dir)
	if err != nil || len(local) == 0 {
		a.StatusWarn("No local branches to open a pull request from")
		return
	}
	base := defaultBaseBranch(remote)
	head := ""
	for _, b := range local {
		if b.Current {
			head = b.Name
		}
	}
	if head == "" {
		head = local[0].Name
	}

	title := widgets.NewInputWidget(widgets.InputConfig{Placeholder: "Title"})
	body := widgets.NewInputWidget(widgets.InputConfig{Placeholder: "Description (optional)"})

	var baseButton, headButton *widgets.ButtonWidget
	baseButton = menuButtonAt("Base: "+base, func(x, y int) {
		a.showNamedMenu("changes.pr.base", remote, base, x, y, func(name string) {
			base = name
			baseButton.SetLabel("Base: " + name)
		})
	})
	headNames := make([]string, 0, len(local))
	for _, b := range local {
		headNames = append(headNames, b.Name)
	}
	headButton = menuButtonAt("Compare: "+head, func(x, y int) {
		a.showNamedMenu("changes.pr.head", headNames, head, x, y, func(name string) {
			head = name
			headButton.SetLabel("Compare: " + name)
		})
	})

	content := widgets.NewVStackWidget(baseButton, headButton, title, body)

	dialog := widgets.NewDialogWidget(60)
	dialog.Title = "New Pull Request"
	dialog.Borders = *a.Borders
	dialog.SetContent(content)
	dialog.Buttons = []widgets.DialogButton{
		{Label: "&Cancel", Handler: func() { a.DismissDialog() }},
		{Label: "&Create", Handler: func() {
			msg := strings.TrimSpace(title.Text())
			if msg == "" {
				a.StatusWarn("A pull request needs a title")
				return
			}
			if base == head {
				a.StatusWarn("Choose two different branches")
				return
			}
			a.DismissDialog()
			a.createPullRequest(dir, base, head, msg, body.Text())
		}},
	}
	dialog.OnDismiss = func() { a.DismissDialog() }
	dialog.Build()

	a.ShowDialog(ui.NewWidgetAdapter(dialog))
}

func (a *App) createPullRequest(dir, base, head, title, description string) {
	var url string
	a.RunRepoTask(RepoTask{
		Progress: "Creating pull request", Done: "Pull request created",
		Dirs: []string{dir},
		Ops:  []RepoOp{OpCreatePullRequest(base, head, title, description, &url)},
		OnDone: func() {
			if url != "" {
				OpenURL(url)
			}
		},
	})
}

// OpenRepoOnGitHub opens the checked-out branch of dir on GitHub.
func (a *App) OpenRepoOnGitHub(dir string) {
	link := git.RepoWebURL(dir, git.BranchName(dir))
	if link == "" {
		a.StatusWarn("No GitHub remote found for this repository")
		return
	}
	OpenURL(link)
}

// defaultBaseBranch prefers the usual trunk names before the first branch.
func defaultBaseBranch(remote []string) string {
	for _, want := range []string{"main", "master"} {
		for _, name := range remote {
			if name == want {
				return want
			}
		}
	}
	return remote[0]
}

// menuButtonAt is a button whose click reports where to open a menu: the spot
// just under the button.
func menuButtonAt(label string, onClick func(x, y int)) *widgets.ButtonWidget {
	var b *widgets.ButtonWidget
	b = widgets.NewButtonWidget(widgets.ButtonConfig{
		Label: label,
		OnClick: func() {
			r := b.GetRect()
			onClick(r.X, r.Y+r.H)
		},
	})
	return b
}
