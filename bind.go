package rtp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/livekit/media-sdk/mixer"
	"github.com/livekit/media-sdk/rtp"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

const (
	printRTCPfromClient = false
)

func (r *Manager) BindRTPtoRoom(
	connRTP, connRTCP net.Conn,
	sID, identity string,
	payloadType byte, clockRate, channels, pTime int,
) error {
	fmt.Printf("BindRTPtoRoom %s: Started binding to session (identity: %s) payload:%d, clockRate:%d, channels:%d, pTime:%d\n",
		sID, identity, payloadType, clockRate, channels, pTime)

	r.mx.Lock()
	defer r.mx.Unlock()

	session, ok := r.session[sID]
	if !ok {
		return fmt.Errorf("session %s: %w", sID, ErrNotFound)
	}

	startedAt := time.Now()

	defer func() {
		duration := time.Since(startedAt)
		fmt.Printf("BindRTPtoRoom %s: Finished (identity: %s) in %v\n", sID, identity, duration)
	}()

	streamRTP := newStreamRTP(connRTP, connRTCP)

	mediaWriter, err := newMediaWriter(rtp.NewSeqWriter(streamRTP), payloadType, clockRate, channels, pTime)
	if err != nil {
		return fmt.Errorf("BindRTPtoRoom %s: failed to create media writer (identity: %s): %w", sID, identity, err)
	}

	rtpProvider, err := newRTPSampleProvider(streamRTP, payloadType, clockRate, channels)
	if err != nil {
		return fmt.Errorf("ACK: failed to create rtpProvider")
	}

	udpConnRTCP := connRTCP.(*net.UDPConn)
	track, err := lksdk.NewLocalTrack(
		webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeOpus,
		},
		lksdk.WithRTCPHandler(func(p rtcp.Packet) {
			data, err := p.Marshal()
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				return
			}

			if err != nil {
				fmt.Printf("track RTCP handler: failed to marshal rtcp packet from LK: %s\n", err)

				return
			}

			rAddrRTCP := streamRTP.GetRemoteAddrRTCP()

			if _, err := udpConnRTCP.WriteTo(data, rAddrRTCP); err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					return
				}

				fmt.Printf("track RTCP handler: failed to write rtcp packet from LK to client: %s\n", err)
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create local track: %w", err)
	}

	mix, err := mixer.NewMixer(
		mediaWriter,
		rtp.DefFrameDur,
		session.stats,
		channels,
		mixer.DefaultInputBufferFrames,
	)
	if err != nil {
		return fmt.Errorf("failed to create mixer: %w", err)
	}

	session.SetParams(
		channels,
		mix,
		track,
		rtpProvider,
		streamRTP,
		connRTCP,
	)

	return nil
}
