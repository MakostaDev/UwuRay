package bittorrent

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/xtls/xray-core/common"
)

type SniffHeader struct{}

func (h *SniffHeader) Protocol() string {
	return "bittorrent"
}

func (h *SniffHeader) Domain() string {
	return ""
}

var errNotBittorrent = errors.New("not bittorrent header")

var bittorrentHandshake = []byte("BitTorrent protocol")

func SniffBittorrent(b []byte) (*SniffHeader, error) {
	if len(b) < 20 {
		return nil, common.ErrNoClue
	}

	if b[0] == 19 && bytes.HasPrefix(b[1:], bittorrentHandshake) {
		return &SniffHeader{}, nil
	}

	return nil, errNotBittorrent
}

func SniffUTP(b []byte) (*SniffHeader, error) {
	if len(b) < 20 {
		return nil, common.ErrNoClue
	}

	// type 4 (ST_SYN), version 1
	if b[0] != 0x41 {
		return nil, errNotBittorrent
	}

	// timestamp_difference is always 0 in new connections
	if binary.BigEndian.Uint32(b[8:12]) != 0 {
		return nil, errNotBittorrent
	}

	// Walk the extension chain. Selective ack (1) and extension bits (2)
	extension, offset := b[1], 20
	for extension != 0 {
		if len(b) < offset+2 {
			return nil, errNotBittorrent
		}
		length := int(b[offset+1])
		switch extension {
		case 1: // selective ack
			if length < 4 || length%4 != 0 {
				return nil, errNotBittorrent
			}
		case 2: // extension bits: fixed 8 bytes, sent in ST_SYN by µTorrent
			if length != 8 {
				return nil, errNotBittorrent
			}
		default:
			return nil, errNotBittorrent
		}
		if len(b) < offset+2+length {
			return nil, errNotBittorrent
		}
		extension = b[offset]
		offset += 2 + length
	}

	// extensions should consume all ST_SYN payload
	if len(b) != offset {
		return nil, errNotBittorrent
	}

	return &SniffHeader{}, nil
}

func SniffUDPTracker(b []byte) (*SniffHeader, error) {
	if len(b) < 16 {
		return nil, common.ErrNoClue
	}

	// protocol_id
	if binary.BigEndian.Uint64(b[0:8]) != 0x41727101980 {
		return nil, errNotBittorrent
	}

	// action connect
	if binary.BigEndian.Uint32(b[8:12]) != 0 {
		return nil, errNotBittorrent
	}

	return &SniffHeader{}, nil
}

var dhtPrefixes = [][]byte{
	[]byte("d1:ad"), // query
	[]byte("d1:rd"), // response
	[]byte("d2:ip"), // BEP-42
	[]byte("d1:el"), // error
}

func SniffDHT(b []byte) (*SniffHeader, error) {
	if len(b) < 5 {
		return nil, common.ErrNoClue
	}

	for _, p := range dhtPrefixes {
		if bytes.HasPrefix(b, p) {
			return &SniffHeader{}, nil
		}
	}

	return nil, errNotBittorrent
}

var lsdPrefix = []byte("BT-SEARCH * HTTP/1.1\r\n")

func SniffLSD(b []byte) (*SniffHeader, error) {
	if len(b) < len(lsdPrefix) {
		return nil, common.ErrNoClue
	}

	if bytes.HasPrefix(b, lsdPrefix) {
		return &SniffHeader{}, nil
	}

	return nil, errNotBittorrent
}
