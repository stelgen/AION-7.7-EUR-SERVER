// admin_text.go — 2007 interactive: ТЕКСТОВЫЙ протокол (L2 C1 канон CAdminSocket.cpp).
//
// КАНОН (src/network/CAdminSocket.cpp, L2CacheD):
//   - OnRead: строки до '\n' (разбиение буфера, Process(command) на каждую строку);
//   - команда: текстовые поля, разделённые табом; ПЕРВОЕ поле = packetId ("d" в
//     ReadSocketArgs), дальше — аргументы ("dsss" у 01_CheckCharacter: charName,
//     adminAccount, unkParam4);
//   - ответ: ТЕКСТ через Send(fmt,...) — например Send("0%d", 3) при пустом charName;
//   - m_timeoutRetries=5, AddTimer(60000) — keepalive 60с.
//
// Aion-вопросы (журнал теорий #14): соответствие interactive packetId ↔ GQ-enum из
// .profile (NPRelay2Server) — ПРИБЛИЖЕНИЕ (обе таблицы = admin-семейство, но канал
// GQ/GP — бинарный NPRelay-релей, отдельный от interactive 2007). Арбитр = capture.
//
// Политика MVP: dispatch по packetId + лог/ship с именем GQ (из профиля), БЕЗ
// текстовых ответов (формы ответов = capture R4; Send("0%d",3)-аналоги после).
package server

import (
	"log"
	"strconv"
	"strings"

	"aion-cached/internal/rpc"
	"aion-cached/internal/ship"
)

// handleAdminText — textline-конвейер 2007 (L2 C1 CAdminSocket канон).
func handleAdminText(ch Channel, name string, lines func() (string, error), respond func(string)) {
	for {
		line, err := lines()
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		id, perr := strconv.Atoi(strings.TrimSpace(fields[0]))
		var opName string
		if perr != nil {
			opName = "<text-no-id>"
		} else {
			// Теория #14: interactive packetId ↔ GQ-enum (NPRelay2Server = idx 6).
			opName = rpc.Name(6, uint16(id))
			if opName == "<UNK>" {
				opName = "GQ_" + strconv.Itoa(id)
			}
		}
		args := fields[1:]
		log.Printf("[cached] admin text: %s args=%d [%s]", opName, len(args),
			strings.Join(clipArgs(args), "|"))
		ship.Emit(ship.Event{Channel: name, Opcode: uint16(max(id, 0)), Name: opName,
			Bytes: len(line), Status: "text"})
		// MVP: текстовых ответов нет (формы = capture R4; L2 C1 канон Send("0%d",3)).
		respond("")
	}
}

// clipArgs — аргументы для лога (макс 4, обрезка 24 символа, сырые тексты —
// данные аккаунтов в лог НЕ пишем полностью: S6-осторожность).
func clipArgs(args []string) []string {
	out := make([]string, 0, 4)
	for i, a := range args {
		if i >= 4 {
			out = append(out, "...")
			break
		}
		if len(a) > 24 {
			a = a[:24] + "…"
		}
		out = append(out, a)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}