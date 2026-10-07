// Package dispatch: снятые дизasmом таблицы ACQ/каналы (dispatch-77.md).
// Нумерация = (slot - base)/8 из ACPacketHandler ctor (VA 0x140078B40).
package dispatch

// Table — команда/имя.
type Entry struct {
	Cmd  uint16
	Name string
}

// T1 — основной диспетчер (Server64/мир -> ACS), cmds 0..39.
// cmd 22 НЕ занят (гэп), 0x28..0x64 = default-резерв.
var T1 = map[uint16]string{
	0x00: "ACQ_UNUSED_0",
	0x01: "ACQ_VERSION_PACKET",
	0x02: "ACQ_TEST_INSERT",
	0x03: "ACQ_TEST_SELECT",
	0x04: "ACQ_FIRST_LOAD_ACCOUNT_INFO",
	0x05: "ACQ_LOAD_BM_PACK_LIST",
	0x06: "ACQ_UPDATE_BM_PACK",
	0x07: "ACQ_LOAD_TRIAL_ACCOUNT_DATA",
	0x08: "ACQ_UPDATE_TRIAL_ACCOUNT_DATA",
	0x09: "ACQ_SYNC_PACKET_TEST",
	0x0a: "ACQ_SAVE_CAHR_CUSTOM",
	0x0b: "ACQ_LOAD_CAHR_CUSTOM",
	0x0c: "ACQ_LOAD_CHAR_CUSTOM_BY_ITEM",
	0x0d: "ACQ_SAVE_CHAR_CUSTOM_TO_ITEM",
	0x0e: "ACQ_LOG_INFO",
	0x0f: "ACQ_CHAR_CREATED",
	0x10: "ACQ_CHAR_LOGIN",
	0x11: "ACQ_CHAR_LOGOUT",
	0x12: "ACQ_CHAR_DELETE",
	0x13: "ACQ_CHAR_DELETE_COMPLETED",
	0x14: "ACQ_CHAR_INFO_REFRESH",
	0x15: "ACQ_CHAR_LEVEL_CHANGED",
	0x17: "ACQ_UPDATE_LOGIN_EVENT_RECORD",
	0x18: "ACQ_DELETE_LOGIN_EVENT_RECORD",
	0x19: "ACQ_UPDATE_HIDDEN_FATIGUE",
	0x1a: "ACQ_LOAD_LUNA",
	0x1b: "ACQ_UPDATE_LUNA",
	0x1c: "ACQ_CONFIRM_LUNA_REWARD",
	0x1d: "ACQ_DECREASE_LUNA_KEY",
	0x1e: "ACQ_UPDATE_BOARD_BM_STATE",
	0x1f: "ACQ_ASK_CAN_MAKE_JUMPING_CHARACTER",
	0x20: "ACQ_LOAD_PREVIOUS_PLAYTIMES_FOR_POLLS",
	0x21: "ACQ_LOAD_PREVIOUS_PLAYTIME_FOR_ONE_POLL",
	0x22: "ACQ_UPDATE_PREVIOUS_PLAYTIME_FOR_ONE_POLL",
	0x23: "ACQ_UPDATE_PREVIOUS_PLAYTIMES_FOR_POLLS",
	0x24: "ACQ_RESET_LUNA_REWARD",
	0x25: "ACQ_TRANSFORM_OPERATION",
	0x26: "ACQ_MONSTER_CORE_UPDATE_VALUE",
	0x27: "ACQ_MONSTER_CORE_UPGRADE",
}

// T2 — второй канал (слоты 0x14449ae38+), семантика уточняется capture'ом (R1).
var T2 = map[uint16]string{
	0x00: "ACQ_GEN_TEST_PACKET1",
	0x01: "ACQ_GEN_TEST_PACKET2",
	0x02: "ACQ_GEN_TEST_PACKET3",
	0x03: "ACQ_MoveCharResultPacket",
	0x04: "ACQ_MoveCharByServicePacket",
	0x05: "ACQ_SetPromotionCoolTimePacket",
	0x06: "ACQ_DeletePromotionCoolTimePacket",
	0x07: "",
}

// Name возвращает имя команды (или "ACQ_UNKNOWN").
func Name(t1 bool, cmd uint16) string {
	if t1 {
		if n, ok := T1[cmd]; ok && n != "" {
			return n
		}
	} else if n, ok := T2[cmd]; ok && n != "" {
		return n
	}
	return "ACQ_UNKNOWN"
}
