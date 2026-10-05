// Package writer — per-service ежедневные файлы в формате оригинального LogServer
// (%s\%04d-%02d-%02d.err), чтобы тейлеры aion-op работали без изменений.
package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type W struct {
	mu   sync.Mutex
	base string              // корень лог-директорий
	dirs map[string]string   // svcType → каталог (сервис-специфичный)
	open map[string]*os.File // "svc|date" → файл
	day  map[string]string
}

func New(base string, dirs map[string]string) *W {
	return &W{base: base, dirs: dirs, open: map[string]*os.File{}, day: map[string]string{}}
}

// DirFor — каталог для типа лога (дефолт: base/svcType).
func (w *W) dirFor(svc string) string {
	if d, ok := w.dirs[svc]; ok && d != "" {
		return d
	}
	return filepath.Join(w.base, svc)
}

// WriteLine — дописать строку в per-service дневной файл (+ .log поток).
// Оригинал: err-файл = %s\%04d-%02d-%02d.err; log = %s\%S\...log (подкаталог).
func (w *W) WriteLine(svc, line string, now time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.writeOne(svc, "err", line, now); err != nil {
		return err
	}
	return w.writeOne(svc, "log", line, now)
}

func (w *W) writeOne(svc, ext, line string, now time.Time) error {
	dir := w.dirFor(svc)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	date := now.Format("2006-01-02")
	key := svc + "|" + ext
	if f := w.open[key]; f != nil && w.day[key] == date {
		return appendLine(f, line)
	}
	_ = w.closeLocked(key)
	fn := filepath.Join(dir, fmt.Sprintf("%s.%s", date, ext))
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", fn, err)
	}
	w.open[key], w.day[key] = f, date
	return appendLine(f, line)
}

func appendLine(f *os.File, line string) error {
	if _, err := f.WriteString(line); err != nil {
		return err
	}
	if len(line) == 0 || line[len(line)-1] != '\n' {
		_, err := f.WriteString("\n")
		return err
	}
	return nil
}

func (w *W) closeLocked(key string) error {
	if f := w.open[key]; f != nil {
		_ = f.Close()
		delete(w.open, key)
	}
	return nil
}

// WriteRaw — отладочный дамп неот разобранных payload (TBD-layout) — отдельный файл.
func (w *W) WriteRaw(svc string, pktType byte, raw []byte, now time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	dir := w.dirFor(svc + ".raw")
	_ = os.MkdirAll(dir, 0o755)
	fn := filepath.Join(dir, now.Format("2006-01-02")+fmt.Sprintf(".t%d.hex", pktType))
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s % X\n", now.Format("15:04:05"), raw)
	return err
}

// WriteStatus — CSV статус-записей (type-5): time,svc,world,x,y,z,engtick,metrics,systime.
func (w *W) WriteStatus(svc string, line string, now time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	dir := w.dirFor(svc)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fn := filepath.Join(dir, now.Format("2006-01-02")+".status.csv")
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s,%s\n", now.Format("15:04:05.000"), line)
	return err
}

// WriteIO — дамп сырых чанков rx/tx до парсинга (capture-истина последней инстанции).
func (w *W) WriteIO(dirTag string, chunk []byte, now time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	dir := w.dirFor("io")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fn := filepath.Join(dir, now.Format("2006-01-02")+".io.hex")
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s %s %d % X\n", now.Format("15:04:05.000"), dirTag, len(chunk), chunk)
	return err
}

// CloseAll — закрыть файлы.
func (w *W) CloseAll() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for k := range w.open {
		_ = w.closeLocked(k)
	}
}
