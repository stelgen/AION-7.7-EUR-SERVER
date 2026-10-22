package rpc

import "testing"

// Тест сгенерированных таблиц: ключевые пары топ-пути + суммарное покрытие.
func TestGeneratedTables(t *testing.T) {
	if ServerDbNames[11] != "RQ_LOAD" || ServerDbNames[108] != "RQ_PING" ||
		ServerDbNames[215] != "RQ_VERSION" {
		t.Fatalf("ServerDb: %q %q %q", ServerDbNames[11], ServerDbNames[108], ServerDbNames[215])
	}
	if DbServerNames[45] != "RP_PING" || DbServerNames[16] != "RP_VERSION" || DbServerNames[1] != "RP_LOAD" {
		t.Fatalf("DbServer: %q %q %q", DbServerNames[45], DbServerNames[16], DbServerNames[1])
	}
	if NprelayServerNames[7] != "GQ_MAKEBUILDER" || NprelayServerNames[3] != "GP_ITEMADD" &&
		ServerNprelayNames[3] != "GP_ITEMADD" {
		// GQ/GP — admin-канал: GQ_MAKEBUILDER критичен для GM-пути
		t.Fatalf("admin GQ/GP mismatch: %q", NprelayServerNames[7])
	}
	total := len(ServerDbNames) + len(DbServerNames) + len(LogServerNames) +
		len(ServerLogNames) + len(IcDbNames) + len(DbIcNames) +
		len(NprelayServerNames) + len(ServerNprelayNames)
	if total != 765 {
		t.Fatalf("total ops = %d, want 765", total)
	}
	if Name(1, 11) != "RQ_LOAD" || Name(0, 9999) != "<UNK>" {
		t.Fatalf("Name(): %q %q", Name(1, 11), Name(0, 9999))
	}
}
