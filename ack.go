package rtp

import (
	"fmt"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

type errCustom string

func (e errCustom) Error() string {
	return string(e)
}

const (
	ErrNotFound = errCustom("not found")
)

func (r *Manager) ACK(sID string, identity string) error {
	session, ok := r.session[sID]
	if !ok {
		fmt.Printf("ACK: track found: %s (identity: %s)\n", sID, identity)

		return fmt.Errorf("track found: %s (identity: %s): %w", sID, identity, ErrNotFound)
	}

	startedAt := time.Now()

	defer func() {
		duration := time.Since(startedAt)
		fmt.Printf("%s: Finished ACK: (identity: %s) in %v\n", sID, identity, duration)
	}()

	track, rtpProvider := session.track, session.rtpProvider

	if _, err := session.room.LocalParticipant.PublishTrack(
		track,
		&lksdk.TrackPublicationOptions{
			Name:   fmt.Sprintf("%s-%d", identity, time.Now().UnixMilli()),
			Stream: track.StreamID(),
		},
	); err != nil {
		return fmt.Errorf("failed to publish track: %w", err)
	}

	fmt.Printf("%s: Started ACK: (identity: %s)\n", sID, identity)

	if err := track.StartWrite(rtpProvider, nil); err != nil {
		return fmt.Errorf("start write to track failed:%s (identity: %s): %w", sID, identity, err)
	}

	return nil
}
