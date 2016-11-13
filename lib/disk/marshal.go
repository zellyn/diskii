// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

// marshal.go contains helpers for marshaling sector structs to/from
// disk.

package disk

// SectorSource is the interface for types that can marshal to sectors.
type SectorSource interface {
	// ToSector marshals the sector struct to exactly 256 bytes.
	ToSector() []byte
	// GetTrack returns the track that a sector struct was loaded from.
	GetTrack() byte
	// GetSector returns the sector that a sector struct was loaded from.
	GetSector() byte
}

// SectorSink is the interface for types that can unmarshal from sectors.
type SectorSink interface {
	// FromSector unmarshals the sector struct from bytes. Input is
	// expected to be exactly 256 bytes.
	FromSector(data []byte)
	// SetTrack sets the track that a sector struct was loaded from.
	SetTrack(track byte)
	// SetSector sets the sector that a sector struct was loaded from.
	SetSector(sector byte)
}

// UnmarshalLogicalSector reads a sector from a SectorDisk, and
// unmarshals it into a SectorSink, setting its track and sector.
func UnmarshalLogicalSector(d LogicalSectorDisk, ss SectorSink, track, sector byte) error {
	bytes, err := d.ReadLogicalSector(track, sector)
	if err != nil {
		return err
	}
	ss.FromSector(bytes)
	ss.SetTrack(track)
	ss.SetSector(sector)
	return nil
}

// MarshalLogicalSector marshals a SectorSource to its sector on a
// SectorDisk.
func MarshalLogicalSector(d LogicalSectorDisk, ss SectorSource) error {
	track := ss.GetTrack()
	sector := ss.GetSector()
	bytes := ss.ToSector()
	return d.WriteLogicalSector(track, sector, bytes)
}
