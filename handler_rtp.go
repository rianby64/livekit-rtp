package rtp

import (
	"fmt"

	"github.com/livekit/media-sdk/rtp"
)

type handlerRTP[T rtp.BytesFrame] struct {
	streamIn *rtp.MediaStreamIn[T]
}

func newHandlerRTP[T rtp.BytesFrame, Sample *rtp.MediaStreamIn[T]](streamIn Sample) rtp.HandlerCloser {
	return &handlerRTP[T]{
		streamIn: streamIn,
	}
}

func (h *handlerRTP[T]) Close() {
	fmt.Printf("handleRTP: Closed\n")
}

func (h *handlerRTP[T]) HandleRTP(header *rtp.Header, payload []byte) error {
	if err := h.streamIn.HandleRTP(header, payload); err != nil {
		return fmt.Errorf("handleRTP: failed to execute HandleRTP: %w", err)
	}

	return nil
}

func (h *handlerRTP[T]) String() string {
	return "bridge handleRTP"
}
