package logger

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"unicode/utf8"
)

// entryEncoder serializes Entry to JSON without encoding/json reflection.
// This eliminates ~67% of allocations that came from reflect.copyVal when
// encoding/json iterates map[string]any via MapIter.Key()/Value().

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func marshalEntry(e *Entry) ([]byte, error) {
	bp := bufPool.Get().(*[]byte)
	buf := (*bp)[:0]

	buf = append(buf, '{')
	buf = appendKey(buf, "timestamp")
	buf = appendString(buf, e.Timestamp)
	buf = append(buf, ',')
	buf = appendKey(buf, "level")
	buf = appendString(buf, e.Level)
	buf = append(buf, ',')
	buf = appendKey(buf, "message")
	buf = appendString(buf, e.Message)
	buf = append(buf, ',')
	buf = appendKey(buf, "service_name")
	buf = appendString(buf, e.ServiceName)

	if e.ServiceVersion != "" {
		buf = append(buf, ',')
		buf = appendKey(buf, "service_version")
		buf = appendString(buf, e.ServiceVersion)
	}
	if e.Environment != "" {
		buf = append(buf, ',')
		buf = appendKey(buf, "environment")
		buf = appendString(buf, e.Environment)
	}
	if e.Caller != "" {
		buf = append(buf, ',')
		buf = appendKey(buf, "caller")
		buf = appendString(buf, e.Caller)
	}
	if len(e.Fields) > 0 {
		buf = append(buf, ',')
		buf = appendKey(buf, "fields")
		var err error
		buf, err = appendMap(buf, e.Fields)
		if err != nil {
			*bp = buf
			bufPool.Put(bp)
			return nil, err
		}
	}
	buf = append(buf, '}')

	out := make([]byte, len(buf))
	copy(out, buf)
	*bp = buf
	bufPool.Put(bp)
	return out, nil
}

func appendKey(buf []byte, key string) []byte {
	buf = append(buf, '"')
	buf = append(buf, key...)
	buf = append(buf, '"', ':')
	return buf
}

func appendString(buf []byte, s string) []byte {
	buf = append(buf, '"')
	buf = appendEscaped(buf, s)
	buf = append(buf, '"')
	return buf
}

func appendEscaped(buf []byte, s string) []byte {
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if b >= 0x20 && b != '"' && b != '\\' {
				buf = append(buf, b)
				i++
				continue
			}
			switch b {
			case '"', '\\':
				buf = append(buf, '\\', b)
			case '\n':
				buf = append(buf, '\\', 'n')
			case '\r':
				buf = append(buf, '\\', 'r')
			case '\t':
				buf = append(buf, '\\', 't')
			default:
				buf = append(buf, '\\', 'u', '0', '0', hexDigit(b>>4), hexDigit(b&0xf))
			}
			i++
		} else {
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 {
				buf = append(buf, '\\', 'u', 'f', 'f', 'f', 'd')
			} else {
				buf = append(buf, s[i:i+size]...)
			}
			i += size
		}
	}
	return buf
}

func hexDigit(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + b - 10
}

func appendMap(buf []byte, m map[string]any) ([]byte, error) {
	buf = append(buf, '{')
	first := true
	for k, v := range m {
		if !first {
			buf = append(buf, ',')
		}
		first = false
		buf = appendKey(buf, k)
		var err error
		buf, err = appendValue(buf, v)
		if err != nil {
			return buf, err
		}
	}
	buf = append(buf, '}')
	return buf, nil
}

func appendValue(buf []byte, v any) ([]byte, error) {
	switch val := v.(type) {
	case nil:
		buf = append(buf, "null"...)
	case string:
		buf = appendString(buf, val)
	case bool:
		if val {
			buf = append(buf, "true"...)
		} else {
			buf = append(buf, "false"...)
		}
	case int:
		buf = strconv.AppendInt(buf, int64(val), 10)
	case int8:
		buf = strconv.AppendInt(buf, int64(val), 10)
	case int16:
		buf = strconv.AppendInt(buf, int64(val), 10)
	case int32:
		buf = strconv.AppendInt(buf, int64(val), 10)
	case int64:
		buf = strconv.AppendInt(buf, val, 10)
	case uint:
		buf = strconv.AppendUint(buf, uint64(val), 10)
	case uint8:
		buf = strconv.AppendUint(buf, uint64(val), 10)
	case uint16:
		buf = strconv.AppendUint(buf, uint64(val), 10)
	case uint32:
		buf = strconv.AppendUint(buf, uint64(val), 10)
	case uint64:
		buf = strconv.AppendUint(buf, val, 10)
	case float32:
		buf = appendFloat(buf, float64(val), 32)
	case float64:
		buf = appendFloat(buf, val, 64)
	case []string:
		buf = append(buf, '[')
		for i, s := range val {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = appendString(buf, s)
		}
		buf = append(buf, ']')
	case []any:
		buf = append(buf, '[')
		for i, item := range val {
			if i > 0 {
				buf = append(buf, ',')
			}
			var err error
			buf, err = appendValue(buf, item)
			if err != nil {
				return buf, err
			}
		}
		buf = append(buf, ']')
	case map[string]any:
		var err error
		buf, err = appendMap(buf, val)
		if err != nil {
			return buf, err
		}
	case error:
		buf = appendString(buf, val.Error())
	case fmt.Stringer:
		buf = appendString(buf, val.String())
	default:
		return buf, fmt.Errorf("unsupported field type %T", v)
	}
	return buf, nil
}

func appendFloat(buf []byte, f float64, bits int) []byte {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return append(buf, "null"...)
	}
	abs := math.Abs(f)
	format := byte('f')
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	buf = strconv.AppendFloat(buf, f, format, -1, bits)
	return buf
}
