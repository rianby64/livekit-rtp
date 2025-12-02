package rtp

import (
	"context"
	"fmt"
	"time"

	"github.com/livekit/media-sdk/g711"
	"github.com/livekit/media-sdk/opus"
	"github.com/livekit/media-sdk/rtp"
	"github.com/pion/webrtc/v4/pkg/media"
	opusv2 "gopkg.in/hraban/opus.v2"
)

const (
	inboundMTU = 1500

	PayloadTypePCMU = 0
	PayloadTypePCMA = 8

	PayloadTypeDynamicStart = 96
	PayloadTypeDynamicEnd   = 127
)

type rtpSampleProvider struct {
	stream      rtp.ReadStream
	header      *rtp.Header
	payload     []byte
	payloadType uint8
	clockRate   int
	encoder     *opusv2.Encoder
}

func newRTPSampleProvider(stream rtp.ReadStream, payloadType uint8, clockRate, channels int) (*rtpSampleProvider, error) {
	encoder, err := opusv2.NewEncoder(clockRate, channels, opusv2.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder in newRTPSampleProvider: %w", err)
	}

	return &rtpSampleProvider{
		stream:      stream,
		header:      &rtp.Header{},
		payload:     make([]byte, inboundMTU),
		payloadType: payloadType,
		clockRate:   clockRate,
		encoder:     encoder,
	}, nil
}

func (r *rtpSampleProvider) Close() error {
	return nil
}

func (s *rtpSampleProvider) NextSample(context.Context) (media.Sample, error) {
	sample := media.Sample{}

	nSamples, err := s.stream.ReadRTP(s.header, s.payload)
	if err != nil {
		return sample, fmt.Errorf("failed to read from RTP socket: %w", err)
	}

	if s.header.PayloadType != s.payloadType {
		fmt.Printf("unexpected payload type: got %d, want %d\n", s.header.PayloadType, s.payloadType)
	}

	sample.Duration = time.Duration(nSamples) * time.Second / time.Duration(s.clockRate)

	switch s.header.PayloadType {
	case PayloadTypePCMA:
		alawSample := make(g711.ALawSample, nSamples)
		copy(alawSample, s.payload)
		pcm := alawSample.Decode()

		numSamplesEncoded, err := s.encoder.Encode(pcm, s.payload)
		if err != nil {
			return sample, fmt.Errorf("failed to encode PCMA(samples=%d) to Opus: %w", nSamples, err)
		}

		sample.Data = s.payload[:numSamplesEncoded]

		return sample, nil

	case PayloadTypePCMU:
		ulawSample := make(g711.ULawSample, nSamples)
		copy(ulawSample, s.payload)
		pcm := ulawSample.Decode()

		numSamplesEncoded, err := s.encoder.Encode(pcm, s.payload)
		if err != nil {
			return sample, fmt.Errorf("failed to encode PCMA(samples=%d) to Opus: %w", nSamples, err)
		}

		sample.Data = s.payload[:numSamplesEncoded]

		return sample, nil

	default:
		if !(PayloadTypeDynamicStart <= s.header.PayloadType && s.header.PayloadType <= PayloadTypeDynamicEnd) {
			return sample, fmt.Errorf("failed to encode PayloadType=%v(samples=%d) to Opus", s.header.PayloadType, nSamples)
		}

		opusSample := make(opus.Sample, nSamples)
		copy(opusSample, s.payload)

		sample.Data = opusSample
	}

	return sample, nil
}

func (r *rtpSampleProvider) OnBind() error {
	return nil
}

func (r *rtpSampleProvider) OnUnbind() error {
	return nil
}
