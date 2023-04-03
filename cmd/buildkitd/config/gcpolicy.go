package config

import (
	"strconv"
	"strings"

	"github.com/docker/go-units"
	"github.com/pkg/errors"
)

type DiskSpace struct {
	Bytes      int64
	Percentage int64
}

var _ encoding.TextUnmarshaler = &DiskSpace{}

func (d *DiskSpace) UnmarshalText(textb []byte) error {
	text := stripQuotes(string(textb))
	if len(text) == 0 {
		return nil
	}

	if text2 := strings.TrimSuffix(text, "%"); len(text2) < len(text) {
		i, err := strconv.ParseInt(text2, 10, 64)
		if err != nil {
			return err
		}
		d.Percentage = i
		return nil
	}

	if i, err := units.RAMInBytes(text); err == nil {
		d.Bytes = i
		return nil
	}

	return errors.Errorf("invalid disk space %s", text)
}

const defaultCap int64 = 2e9 // 2GB

func DefaultGCPolicy(keep DiskSpace) []GCPolicy {
	if keep == (DiskSpace{}) {
		keep = DetectDefaultGCCap()
	}
	return []GCPolicy{
		// if build cache uses more than 512MB delete the most easily reproducible data after it has not been used for 2 days
		{
			Filters:      []string{"type==source.local,type==exec.cachemount,type==source.git.checkout"},
			KeepDuration: 48 * 3600,                   // 48h
			KeepBytes:    DiskSpace{Bytes: 512 * 1e6}, // 512MB
		},
		// remove any data not used for 60 days
		{
			KeepDuration: 60 * 24 * 3600, // 60d
			KeepBytes:    keep,
		},
		// keep the unshared build cache under cap
		{
			KeepBytes: keep,
		},
		// if previous policies were insufficient start deleting internal data to keep build cache under cap
		{
			All:       true,
			KeepBytes: keep,
		},
	}
}

func stripQuotes(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
