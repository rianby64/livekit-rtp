package rtp

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/livekit/media-sdk/opus"
	"github.com/livekit/media-sdk/rtp"
	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

func (r *Manager) subscribeTrack(sID, identity string) func(*webrtc.TrackRemote, *lksdk.RemoteTrackPublication, *lksdk.RemoteParticipant) {
	return func(track *webrtc.TrackRemote, rTrack *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		fmt.Printf("%s: Started OnTrackSubscribed: %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)

		defer fmt.Printf("%s: Finished OnTrackSubscribed: %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)

		session, ok := r.session[sID]
		if !ok {
			fmt.Printf("OnTrackSubscribed: session %s not found %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)

			return
		}

		mixer := session.mixer
		if mixer == nil {
			fmt.Printf("OnTrackSubscribed: mixer in session %s not ready %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)

			return
		}

		connRTCP := session.connRTCP
		if connRTCP == nil {
			fmt.Printf("OnTrackSubscribed: connRTCP in session %s not ready %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)

			return
		}

		streamRTP := session.streamRTP
		if streamRTP == nil {
			fmt.Printf("OnTrackSubscribed: streamRTP in session %s not ready %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)

			return
		}

		mTrack := mixer.NewInput()

		defer func() {
			if err := mTrack.Close(); err != nil {
				fmt.Printf("OnTrackSubscribed: failed to close mTrack %s %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)
			}

			mixer.RemoveInput(mTrack)
		}()

		decoder, err := opus.Decode(mTrack, session.channels, logger.GetLogger())
		if err != nil {
			fmt.Printf("OnTrackSubscribed: failed to create decoder in session %s %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)

			return
		}

		handler := newHandlerRTP(rtp.NewMediaStreamIn(decoder))

		defer handler.Close()

		handlerJitter := rtp.HandleJitter(handler)

		defer handlerJitter.Close()

		if err := rtp.HandleLoop(track, handlerJitter); err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				return
			}

			fmt.Printf("OnTrackSubscribed: session %s failed to exit HandleLoop %s (identity: %s ?== %s)\n", sID, track.ID(), rp.Identity(), identity)
		}
	}
}
