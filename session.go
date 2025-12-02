package rtp

import (
	"net"

	"github.com/livekit/media-sdk/mixer"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type session struct {
	room        *lksdk.Room
	stats       *mixer.Stats
	mixer       *mixer.Mixer
	track       *lksdk.LocalTrack
	rtpProvider *rtpSampleProvider
	streamRTP   *streamRTP
	connRTCP    net.Conn

	channels int
}

func newSession(room *lksdk.Room) *session {
	return &session{
		room:     room,
		stats:    &mixer.Stats{},
		channels: 1,
	}
}

func (s *session) SetParams(
	channels int,
	mixer *mixer.Mixer,
	track *lksdk.LocalTrack,
	rtpProvider *rtpSampleProvider,
	streamRTP *streamRTP,
	connRTCP net.Conn,
) {
	s.channels,
		s.mixer,
		s.track,
		s.rtpProvider,
		s.streamRTP,
		s.connRTCP = channels,
		mixer,
		track,
		rtpProvider,
		streamRTP,
		connRTCP
}
