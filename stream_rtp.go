package rtp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/livekit/media-sdk/rtp"
	"github.com/pion/rtcp"
)

const (
	deadlineUDP = time.Minute
)

type streamRTP struct {
	connRTP, connRTCP *net.UDPConn
	buff              []byte

	rAddrRTPWait chan struct{}
	rAddrRTPMx   sync.Mutex
	rAddrRTP     net.Addr

	rAddrRTCPWait chan struct{}
	rAddrRTCPMx   sync.Mutex
	rAddrRTCP     net.Addr

	closed atomic.Bool

	rtpBuff chan rtp.Packet
}

func shouldExit(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}

	if errNet, ok := err.(net.Error); ok && errNet.Timeout() {
		return true
	}

	return false
}

func newStreamRTP(connRTP, connRTCP net.Conn) *streamRTP {
	udpConnRTP := connRTP.(*net.UDPConn)
	udpConnRTCP := connRTCP.(*net.UDPConn)

	c := &streamRTP{
		connRTP:       udpConnRTP,
		connRTCP:      udpConnRTCP,
		buff:          make([]byte, inboundMTU),
		rtpBuff:       make(chan rtp.Packet, 65535),
		rAddrRTPWait:  make(chan struct{}, 1),
		rAddrRTCPWait: make(chan struct{}, 1),
	}

	go func() {
		buff := make([]byte, inboundMTU)

		for {
			if err := udpConnRTCP.SetDeadline(time.Now().Add(deadlineUDP)); err != nil {
				if shouldExit(err) {
					return
				}

				fmt.Printf("Error setting the deadline for UDP: %v\n", err)

				continue
			}

			n, rAddr, err := udpConnRTCP.ReadFromUDP(buff)
			if err != nil {
				if shouldExit(err) {
					fmt.Printf("RTCP connection closed, stopping read loop\n")

					return
				}

				// close even if Deadline has been exceeded
				fmt.Println("RTCP read error:", err)

				return
			}

			c.SetRemoteAddrRTCP(rAddr)

			pkts, err := rtcp.Unmarshal(buff[:n])
			if err != nil {
				fmt.Println("streamRTP: RTCP unmarshal error:", err)

				continue
			}

			if !printRTCPfromClient {
				continue
			}

			for _, p := range pkts {
				fmt.Printf("Got RTCP from %s: %+v\n", rAddr, p)
			}
		}
	}()

	go func() {
		defer close(c.rtpBuff)

		for {
			if err := udpConnRTP.SetDeadline(time.Now().Add(deadlineUDP)); err != nil {
				if shouldExit(err) {
					return
				}

				fmt.Printf("Error setting the deadline for UDP: %v\n", err)

				continue
			}

			n, rAddr, err := udpConnRTP.ReadFromUDP(c.buff)
			if err != nil {
				if shouldExit(err) {
					fmt.Println("RTP connection closed, stopping read loop")

					return
				}

				fmt.Printf("streamRTP: ReadRTP failed: %s\n", err)

				// close even if Deadline has been exceeded
				return
			}

			c.SetRemoteAddrRTP(rAddr)

			pkt := rtp.Packet{}

			if err := pkt.Unmarshal(c.buff[:n]); err != nil {
				fmt.Printf("streamRTP: RTP unmarshal error: %s\n", err)

				continue
			}

			c.rtpBuff <- pkt
		}
	}()

	return c
}

func (c *streamRTP) Close() {
	if c.closed.Swap(true) {
		fmt.Printf("streamRTP already closed\n")

		return
	}

	if err := c.connRTP.Close(); err != nil {
		fmt.Printf("failed to close RTP conn: %v\n", err)
	}

	if err := c.connRTCP.Close(); err != nil {
		fmt.Printf("failed to close RTCP conn: %v\n", err)
	}
}

func (c *streamRTP) String() string {
	return "stream RTP"
}

func (c *streamRTP) WriteRTP(h *rtp.Header, payload []byte) (int, error) {
	rAddr := c.GetRemoteAddrRTP()

	pkt := rtp.Packet{
		Header:  *h,
		Payload: payload,
	}

	data, err := pkt.Marshal()
	if err != nil {
		return 0, fmt.Errorf("streamRTP: failed to marshal pkt: %w", err)
	}

	n, err := c.connRTP.WriteTo(data, rAddr)
	if err != nil {
		// close even if Deadline has been exceeded
		return n, fmt.Errorf("streamRTP: failed to write data: %w", err)
	}

	return n, nil
}

func (c *streamRTP) GetRemoteAddrRTP() net.Addr {
	<-c.rAddrRTPWait

	c.rAddrRTPMx.Lock()
	defer c.rAddrRTPMx.Unlock()

	return c.rAddrRTP
}

func (c *streamRTP) SetRemoteAddrRTP(addr net.Addr) {
	c.rAddrRTPMx.Lock()
	defer c.rAddrRTPMx.Unlock()

	if c.rAddrRTP == nil {
		defer close(c.rAddrRTPWait)
	}

	c.rAddrRTP = addr
}

func (c *streamRTP) GetRemoteAddrRTCP() net.Addr {
	<-c.rAddrRTCPWait

	c.rAddrRTCPMx.Lock()
	defer c.rAddrRTCPMx.Unlock()

	return c.rAddrRTCP
}

func (c *streamRTP) SetRemoteAddrRTCP(addr net.Addr) {
	c.rAddrRTCPMx.Lock()
	defer c.rAddrRTCPMx.Unlock()

	if c.rAddrRTCP == nil {
		defer close(c.rAddrRTCPWait)
	}

	c.rAddrRTCP = addr
}

func (c *streamRTP) ReadRTP(h *rtp.Header, payload []byte) (int, error) {
	pkt, ok := <-c.rtpBuff
	if !ok {
		return 0, io.EOF
	}

	copy(payload, pkt.Payload)
	*h = pkt.Header

	return len(pkt.Payload), nil
}
