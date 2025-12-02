package rtp

import (
	"net"

	"github.com/livekit/media-sdk/mixer"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/livekit/sip/pkg/sip"
)

type session struct {
	room        *lksdk.Room
	roomStats   *sip.RoomStats
	mixer       *mixer.Mixer
	track       *lksdk.LocalTrack
	rtpProvider *rtpSampleProvider
	streamRTP   *streamRTP
	connRTCP    net.Conn

	channels int
}

func newSession(room *lksdk.Room) *session {
	return &session{
		room:      room,
		roomStats: &sip.RoomStats{},
		channels:  1,
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
