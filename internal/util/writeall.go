package util

import (
	"io"
)

func WriteAll(w io.Writer, data []byte) (int, error) {
	remain := data
	for len(remain) > 0 {
		n, err := w.Write(remain)
		remain = remain[n:]
		if err != nil {
			return len(data) - len(remain), err
		}
	}
	return len(data), nil
}
