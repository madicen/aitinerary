// Package mouse owns the bubblezone ID catalog for the aitinerary TUI.
// Centralising IDs prevents drift between zone.Mark and zone.Get call sites.
package mouse

import "fmt"

const (
	// Tab bar zones.
	ZoneTabBoard    = "zone:tab:board"
	ZoneTabQueue    = "zone:tab:queue"
	ZoneTabEpics    = "zone:tab:epics"
	ZoneTabSettings = "zone:tab:settings"
	ZoneTabHelp     = "zone:tab:help"

	// Status bar action zones.
	ZoneStatusHelp    = "zone:status:help"
	ZoneStatusPalette = "zone:status:palette"
	ZoneStatusRefresh = "zone:status:refresh"
	ZoneStatusUndo    = "zone:status:undo"
	ZoneStatusQuit    = "zone:status:quit"

	// Board filter chips.
	ZoneFilterAll             = "zone:board:filter:all"
	ZoneFilterNeedsAttention  = "zone:board:filter:needs_attention"
	ZoneFilterDoneCandidates  = "zone:board:filter:done_candidates"
	ZoneFilterWorkspacePrefix = "zone:board:filter:workspace:"
	ZoneFilterStatusPrefix    = "zone:board:filter:status:"

	// Board detail edit affordances.
	ZoneBoardDetailSprint     = "zone:board:detail:sprint"
	ZoneBoardDetailDueDate    = "zone:board:detail:duedate"
	ZoneBoardDetailParent     = "zone:board:detail:parent"
	ZoneBoardDetailLabels     = "zone:board:detail:labels"
	ZoneBoardDetailTransition = "zone:board:detail:transition"

	// Board row prefix — indexed by list position.
	ZoneBoardRowPrefix = "zone:board:row:"

	// Overlay zones.
	ConfirmYes   = "zone:confirm:yes"
	ConfirmNo    = "zone:confirm:no"
	ErrorDismiss = "zone:error:dismiss"
	ErrorCopy    = "zone:error:copy"
	HelpDismiss  = "zone:help:dismiss"
	SetupWizardDismiss = "zone:setup:dismiss"
	PaletteRowPrefix = "zone:palette:row:"
)

// ZoneBoardRow returns the zone ID for a board list row at index i.
func ZoneBoardRow(i int) string {
	return fmt.Sprintf("%s%d", ZoneBoardRowPrefix, i)
}

// ZoneFilterWorkspace returns the zone ID for a workspace filter chip.
func ZoneFilterWorkspace(name string) string {
	return ZoneFilterWorkspacePrefix + name
}

// ZoneFilterStatus returns the zone ID for a status-category filter chip.
func ZoneFilterStatus(category string) string {
	return ZoneFilterStatusPrefix + category
}

// PaletteRow marks one command row in the ctrl+k palette overlay.
func PaletteRow(i int) string {
	return fmt.Sprintf("%s%d", PaletteRowPrefix, i)
}

// ZoneGenMenuItemPrefix is the prefix for long-press generate-button menu rows.
const ZoneGenMenuItemPrefix = "zone:genmenu:"

// ZoneGenMenuItem returns the zone ID for a genmenu row at index i.
func ZoneGenMenuItem(i int) string {
	return fmt.Sprintf("%s%d", ZoneGenMenuItemPrefix, i)
}

// ZoneBoardCtxMenuItem returns the zone ID for a board context menu item.
func ZoneBoardCtxMenuItem(i int) string {
	return fmt.Sprintf("zone:board:ctxmenu:%d", i)
}

// Queue tab zones.
const (
	ZoneQueueRowPrefix    = "zone:queue:row:"
	ZoneQueueApprovePrefix = "zone:queue:approve:"
	ZoneQueueEditPrefix   = "zone:queue:edit:"
	ZoneQueueRejectPrefix = "zone:queue:reject:"
	ZoneQueueBulkApprove  = "zone:queue:bulk_approve"
)

// ZoneQueueRow returns the zone ID for a queue list row at flat index i.
func ZoneQueueRow(i int) string { return fmt.Sprintf("%s%d", ZoneQueueRowPrefix, i) }

// ZoneQueueApprove returns the zone ID for the approve control on row i.
func ZoneQueueApprove(i int) string { return fmt.Sprintf("%s%d", ZoneQueueApprovePrefix, i) }

// ZoneQueueEdit returns the zone ID for the edit control on row i.
func ZoneQueueEdit(i int) string { return fmt.Sprintf("%s%d", ZoneQueueEditPrefix, i) }

// ZoneQueueReject returns the zone ID for the reject control on row i.
func ZoneQueueReject(i int) string { return fmt.Sprintf("%s%d", ZoneQueueRejectPrefix, i) }

// Epics tab zones.
const (
	ZoneEpicsRowPrefix     = "zone:epics:row:"
	ZoneEpicsTimeline      = "zone:epics:timeline"
	ZoneEpicsTimelineBarPrefix = "zone:epics:timeline:bar:"
)

// ZoneEpicsRow returns the zone ID for an epics list row at index i.
func ZoneEpicsRow(i int) string { return fmt.Sprintf("%s%d", ZoneEpicsRowPrefix, i) }

// ZoneEpicsTimelineBar returns the zone ID for a timeline bar at index i.
func ZoneEpicsTimelineBar(i int) string {
	return fmt.Sprintf("%s%d", ZoneEpicsTimelineBarPrefix, i)
}

// Board generate + sync-prep zones.
const (
	ZoneBoardGenerate = "zone:board:generate"
	ZoneSweepCancel   = "zone:sweep:cancel"
	ZoneSweepDismiss  = "zone:sweep:dismiss"
)

// Status bar queue badge.
const ZoneStatusQueue = "zone:status:queue"
