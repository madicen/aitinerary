package board

import (
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/tui/mouse"
)

func (m *Model) zoneIDs() []string {
	ids := []string{
		mouse.ZoneFilterAll,
		mouse.ZoneFilterNeedsAttention,
		mouse.ZoneFilterDoneCandidates,
		mouse.ZoneBoardDetailSprint,
		mouse.ZoneBoardDetailDueDate,
		mouse.ZoneBoardDetailParent,
		mouse.ZoneBoardDetailLabels,
		mouse.ZoneBoardDetailTransition,
	}
	for _, ws := range m.chips {
		ids = append(ids, mouse.ZoneFilterWorkspace(ws.Chip))
	}
	for _, cat := range []string{"new", "indeterminate", "done"} {
		ids = append(ids, mouse.ZoneFilterStatus(cat))
	}
	for i := 0; i < m.visibleTicketCount(); i++ {
		ids = append(ids, mouse.ZoneBoardRow(i))
	}
	for i := range contextMenuItems() {
		ids = append(ids, mouse.ZoneBoardCtxMenuItem(i))
	}
	return ids
}

func (m *Model) resolveClickedZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil || m.zone == nil {
		return ""
	}
	for _, id := range m.zoneIDs() {
		if z := m.zone.Get(id); z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}

// zonePressGen tracks left-press dedupe for overlapping zones.
var zonePressGen uint64

func onZonePress() { zonePressGen++ }
