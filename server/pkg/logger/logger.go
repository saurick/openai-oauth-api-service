// 日志输出，支持颜色
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/jwalton/gchalk"
	"github.com/jwalton/go-supportscolor"
)

var _ log.Logger = (*stdColorLogger)(nil)

type Logger interface {
	WithContext(ctx context.Context) *log.Helper
}

// stdColorLogger corresponds to the standard library's [log.Logger] and provides
// similar capabilities. It also can be used concurrently by multiple goroutines.
type stdColorLogger struct {
	w         io.Writer
	debug     bool
	skipN     bool
	isDiscard bool
	mu        sync.Mutex
	pool      *sync.Pool
}

// 带颜色输出的 logger
func NewStdColorLogger(w io.Writer, skipNullValue, debug bool) log.Logger {
	return &stdColorLogger{
		w:         w,
		debug:     debug,
		skipN:     skipNullValue, // 跳过空值不输出
		isDiscard: w == io.Discard,
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Log print the kv pairs log.
func (l *stdColorLogger) Log(level log.Level, keyvals ...interface{}) error {
	// if disable debug, skip
	if level == log.LevelDebug && !l.debug {
		return nil
	}
	if l.isDiscard || len(keyvals) == 0 {
		return nil
	}
	if (len(keyvals) & 1) == 1 {
		keyvals = append(keyvals, "KEYVALS UNPAIRED")
	}

	// wirter 不支持颜色输出(线上环境)，直接输出 json
	if w, ok := l.w.(*os.File); !ok || supportscolor.SupportsColor(w.Fd()).Level == gchalk.LevelNone {
		testMode := flag.Lookup("test.v") != nil // 是否是单元测试环境
		if !testMode {                           // 特殊的，单元测试环境下，不输出 json 格式
			return l.jsonOutput(level, keyvals...)
		}
	}

	buf := l.pool.Get().(*bytes.Buffer)
	defer l.pool.Put(buf)

	title := level.String()
	color := func(str ...string) string { return gchalk.Gray(str...) }
	switch level {
	case log.LevelDebug:
		title = gchalk.Green(title)
		color = gchalk.Green
	case log.LevelInfo:
		title = gchalk.Blue(title)
		color = gchalk.Blue
	case log.LevelWarn:
		title = gchalk.Yellow(title)
		color = gchalk.Yellow
	case log.LevelError:
		title = gchalk.BgBrightRed(title)
		color = gchalk.BgBrightRed
	case log.LevelFatal:
		title = gchalk.BgBrightRed(title)
		color = gchalk.BgBrightRed
	}
	buf.WriteString(color(title))

	for i := 0; i < len(keyvals); i += 2 {
		k := fmt.Sprintf("%s", keyvals[i])
		v := fmt.Sprintf("%v", keyvals[i+1])

		// 跳过空值不输出
		if l.skipN && v == "" {
			continue
		}

		// caller字段加个空格，方便 VSCode 编辑器点击跳转到代码
		if l.debug && k == "caller" {
			v = " " + v
		}

		_, _ = fmt.Fprintf(buf, " %s%s%v", gchalk.Gray(k), gchalk.Gray("="), v)
	}
	buf.WriteByte('\n')
	defer buf.Reset()

	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.w.Write(buf.Bytes())
	return err
}

func (l *stdColorLogger) jsonOutput(level log.Level, keyvals ...interface{}) error {
	param := map[string]interface{}{"level": level.String()}
	for i := 0; i < len(keyvals); i += 2 {
		k := fmt.Sprintf("%v", keyvals[i])
		v := keyvals[i+1]
		param[k] = v
	}
	data, err := json.Marshal(&param)
	if err != nil {
		return err
	}
	_, err = l.w.Write(data)
	if err != nil {
		return err
	}
	_, err = l.w.Write([]byte("\n"))
	if err != nil {
		return err
	}
	return nil
}

func (l *stdColorLogger) Close() error {
	return nil
}
