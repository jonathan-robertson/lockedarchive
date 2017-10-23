package stream

import (
	"context"
	"io"
)

// GetChunk reads from r until chunk has been filled or until EOF is reached
func GetChunk(ctx context.Context, r io.Reader, chunk []byte) (length int, err error) {
	singleByte := make([]byte, 1)

	// Loop through bytes until chunk fills or EOF is reached
	for i := 0; i < len(chunk); i++ {
		select {
		case <-ctx.Done():
			return 0, context.Canceled

		default:
			if _, err = r.Read(singleByte); err != nil {
				return // return even if io.EOF
			}

			// Write received byte to chunk
			// TODO: Is it more effective for us to append bytes and return chunk?
			chunk[i] = singleByte[0]
			length++
		}
	}

	return
}
