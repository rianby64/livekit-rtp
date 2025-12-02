package rtp

import (
	"fmt"

	"github.com/livekit/media-sdk"
	"github.com/livekit/media-sdk/g711"
	"github.com/livekit/media-sdk/opus"
	"github.com/livekit/media-sdk/rtp"
	"github.com/livekit/protocol/logger"
)

type rtpWriteSample[
	S g711.ALawSample |
		g711.ULawSample |
		opus.Sample,
] struct {
	rtpWriter *rtp.Stream
	clockRate int
	marker    bool
}

func newRTPWriteSample[
	S g711.ALawSample |
		g711.ULawSample |
		opus.Sample,
](clockRate int, rtpWriter *rtp.Stream) *rtpWriteSample[S] {
	return &rtpWriteSample[S]{
		rtpWriter: rtpWriter,
		clockRate: clockRate,
		marker:    true,
	}
}

func (w *rtpWriteSample[S]) Close() error {
	return nil
}

func (w *rtpWriteSample[S]) SampleRate() int {
	return w.clockRate
}

func (w *rtpWriteSample[S]) String() string {
	return "rtp-write-sample"
}

func (w *rtpWriteSample[S]) WriteSample(sample S) error {
	if err := w.rtpWriter.WritePayload([]byte(sample), w.marker); err != nil {
		return fmt.Errorf("rtpWriteSample[%T]: failed to write payload (sample=%d):%w", sample, len(sample), err)
	}

	w.marker = false

	return nil
}

type mediaWriter[Writer media.Writer[media.PCM16Sample]] struct {
	encoder   media.PCM16Writer
	clockRate int
}

func newMediaWriter(seqWriter *rtp.SeqWriter, payloadType byte, clockRate, channels, ptime int) (*mediaWriter[media.Writer[media.PCM16Sample]], error) {
	rtpWriter := seqWriter.NewStreamWithDur(payloadType, uint32(clockRate*ptime/1000))

	var encoder media.PCM16Writer

	switch payloadType {
	case PayloadTypePCMA:
		encoder = g711.EncodeALaw(newRTPWriteSample[g711.ALawSample](clockRate, rtpWriter))
	case PayloadTypePCMU:
		encoder = g711.EncodeULaw(newRTPWriteSample[g711.ULawSample](clockRate, rtpWriter))
	default:
		if !(PayloadTypeDynamicStart <= payloadType && payloadType <= PayloadTypeDynamicEnd) {
			return nil, fmt.Errorf("unsupported payload type: %d", payloadType)
		}

		opusEncoder, err := opus.Encode(
			newRTPWriteSample[opus.Sample](clockRate, rtpWriter),
			channels, logger.GetLogger(),
		)

		if err != nil {
			return nil, fmt.Errorf("cannot create opus encoder: %w", err)
		}

		encoder = opusEncoder
	}

	return &mediaWriter[media.Writer[media.PCM16Sample]]{
		encoder:   encoder,
		clockRate: clockRate,
	}, nil
}

func (m *mediaWriter[Writer]) SampleRate() int {
	return m.clockRate
}

func (m *mediaWriter[Writer]) String() string {
	return "custom media writer"
}

func (m *mediaWriter[Writer]) WriteSample(sample media.PCM16Sample) error {
	if err := m.encoder.WriteSample(sample); err != nil {
		return fmt.Errorf("custom media writer: failed to write sample: %w", err)
	}

	return nil
}
