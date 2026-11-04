package app

import (
	"testing"

	"github.com/eugenioenko/ttt/internal/widgets"
)

func syncRowButtons(t *testing.T, cp *ChangesPanel) []*widgets.ButtonWidget {
	t.Helper()
	var buttons []*widgets.ButtonWidget
	for _, child := range cp.syncRow.Children {
		b, ok := child.(*widgets.ButtonWidget)
		if !ok {
			t.Fatalf("sync row child is %T, want *widgets.ButtonWidget", child)
		}
		buttons = append(buttons, b)
	}
	return buttons
}

func syncRowLabels(t *testing.T, cp *ChangesPanel) []string {
	t.Helper()
	var labels []string
	for _, b := range syncRowButtons(t, cp) {
		labels = append(labels, b.Config.Label)
	}
	return labels
}

func syncRowSameLabels(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSyncRowWithoutRepoIsHidden(t *testing.T) {
	cp := NewChangesPanel()
	cp.rebuildSyncRow()
	if len(cp.syncRow.Children) != 0 || cp.syncRow.FixedHeight != 0 {
		t.Fatalf("sync row with no repo: children=%d height=%d, want none", len(cp.syncRow.Children), cp.syncRow.FixedHeight)
	}
}

func TestSyncRowShowsFetchAndPushWhenOriginHasNoNewCommits(t *testing.T) {
	cp := NewChangesPanel()
	cp.groups = []changesGroup{{Dir: "/repo", Upstream: true, Ahead: 1}}
	cp.rebuildSyncRow()

	if want := []string{"Fetch", "Push ↑1"}; !syncRowSameLabels(syncRowLabels(t, cp), want) {
		t.Fatalf("labels = %q, want %q", syncRowLabels(t, cp), want)
	}
}

func TestSyncRowAddsPullWhenOriginIsAhead(t *testing.T) {
	cp := NewChangesPanel()
	cp.groups = []changesGroup{{Dir: "/repo", Upstream: true, Behind: 2, Ahead: 1}}
	cp.rebuildSyncRow()

	if want := []string{"Fetch", "Pull ↓2", "Push ↑1"}; !syncRowSameLabels(syncRowLabels(t, cp), want) {
		t.Fatalf("labels = %q, want %q", syncRowLabels(t, cp), want)
	}
}

func TestSyncRowWithoutUpstreamOffersPushButNoPull(t *testing.T) {
	cp := NewChangesPanel()
	cp.groups = []changesGroup{{Dir: "/repo"}}
	cp.rebuildSyncRow()

	if want := []string{"Fetch", "Push"}; !syncRowSameLabels(syncRowLabels(t, cp), want) {
		t.Fatalf("labels = %q, want %q", syncRowLabels(t, cp), want)
	}
}

func TestSyncRowButtonsReportTheirOperation(t *testing.T) {
	cp := NewChangesPanel()
	cp.groups = []changesGroup{{Dir: "/repo", Upstream: true, Behind: 1}}
	cp.rebuildSyncRow()

	var got []string
	cp.OnSync = func(dir string, op RepoOp) {
		if dir != "/repo" {
			t.Errorf("OnSync dir = %q, want /repo", dir)
		}
		got = append(got, op.Name)
	}
	for _, b := range syncRowButtons(t, cp) {
		b.Config.OnClick()
	}
	if want := []string{"fetch", "pull", "push"}; !syncRowSameLabels(got, want) {
		t.Fatalf("operations = %q, want %q", got, want)
	}
}
