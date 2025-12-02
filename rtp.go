package rtp

import (
	"fmt"
	"sync"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

type ConfigLK struct {
	LivekitApiKey    string
	LivekitApiSecret string
	LivekitUrl       string
}

type Manager struct {
	mx sync.Mutex

	session map[string]*session

	config *ConfigLK
}

func NewManager(config *ConfigLK) *Manager {
	return &Manager{
		session: make(map[string]*session),

		config: config,
	}
}

func (r *Manager) ConnectToRoom(roomName, user, identity string) (string, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	timestamp := time.Now().UnixNano()

	sID := fmt.Sprintf("%s-%s-%d", roomName, identity, timestamp)

	if _, ok := r.session[sID]; ok {
		fmt.Printf("already connected to room: %s (%s:%s)\n", roomName, user, identity)

		return sID, nil
	}

	startedAt := time.Now()

	defer func() {
		duration := time.Since(startedAt)
		fmt.Printf("%s: Finished connecting to Livekit room: %s as user: %s (identity: %s) in %v\n",
			sID, roomName, user, identity, duration,
		)
	}()

	fmt.Printf("%s: Started ConnectToRoom: %s identity: %s\n", sID, roomName, identity)

	cb := lksdk.NewRoomCallback()
	cb.OnTrackSubscribed = r.subscribeTrack(sID, identity)

	room, err := lksdk.ConnectToRoom(r.config.LivekitUrl,
		lksdk.ConnectInfo{
			APIKey:              r.config.LivekitApiKey,
			APISecret:           r.config.LivekitApiSecret,
			RoomName:            roomName,
			ParticipantName:     user,
			ParticipantIdentity: fmt.Sprintf("%s-%d", identity, timestamp),
			ParticipantKind:     lksdk.ParticipantSIP,
		},
		cb,
	)
	if err != nil {
		return "", fmt.Errorf("failed to connect to room: %w", err)
	}

	r.session[sID] = newSession(room)

	return sID, nil
}

func (r *Manager) DisconnectFromRoom(sID string) error {
	if sID == "" {
		return nil
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	session, ok := r.session[sID]
	if !ok {
		return fmt.Errorf("session %s: %w", sID, ErrNotFound)
	}

	defer delete(r.session, sID)

	if session.streamRTP != nil {
		session.streamRTP.Close()
	}

	if session.mixer != nil {
		session.mixer.Stop()
	}

	if session.track != nil {
		if err := session.track.Close(); err != nil {
			fmt.Printf("DisconnectFromRoom: failed to close track %s: %v\n", session.track.ID(), err)
		}
	}

	if session.room != nil {
		session.room.Disconnect()
	}

	return nil
}
