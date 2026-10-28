// Package rpc — R5 #8 v20: RP_VERSION (O16, 27Б) мир-init формы.
package rpc

// o16Version — capture 24.10 (cdmirror-20261024-worldstart.log, 02:44:33,
// ответ на C215 от Server64/NPCSvr): [len=27][cmd=16][0xAF][~0xFFEF]
// payload = 9c0f0300 7c000000 d8000000 00008000 00000000.
// СМЫСЛ: CacheD «version», Server64 ждёт её на «Waiting for Cache Version».
var o16Version = []byte{
	0x1b, 0x00, 0x10, 0x00, 0xaf, 0xef, 0xff,
	0x9c, 0x0f, 0x03, 0x00,
	0x7c, 0x00, 0x00, 0x00,
	0xd8, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x80, 0x00,
	0x00, 0x00, 0x00, 0x00,
}

// O16VersionFrame — константный кадр RP_VERSION (27Б, capture-эталон).
func O16VersionFrame() []byte {
	out := make([]byte, len(o16Version))
	copy(out, o16Version)
	return out
}
