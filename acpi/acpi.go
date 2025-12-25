// Package acpi is used to leverage the ACPI standard, especially by focusing
// on parsing the Firmware Performance Data Table to retrieve boot metrics.
package acpi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	tableHeaderSize   int    = 36
	pathDevMem        string = "/dev/mem"
	pathFPDTBootDir   string = "/sys/firmware/acpi/fpdt/boot/"
	pathFPDTTableFile string = "/sys/firmware/acpi/tables/FPDT"
)

// TableHeader is the standard header common to all ACPI tables (36 bytes).
type TableHeader struct {
	// Signature is a a 4-byte slice identifying the table ("ECDT", "FPDT", etc).
	Signature       [4]byte
	Length          uint32
	Revision        uint8
	Checksum        uint8
	OEMID           [6]byte
	OEMTableID      [8]byte
	OEMRevision     uint32
	CreatorID       uint32
	CreatorRevision uint32
}

// TableHeaderFPDT is the common header for FPDT records inside.
type TableHeaderFPDT struct {
	Type     uint16
	Length   uint8
	Revision uint8
}

// TablePointerRecordFPDT is a pointer to the FPDT values.
type TablePointerRecordFPDT struct {
	Header   TableHeaderFPDT
	Reserved uint32
	// Address is the physical address of the the FPDT table in memory.
	Address uint64
}

// TableRecordFPDT is the content of the FPDT table.
type TableRecordFPDT struct {
	// Header is the header of the table.
	Header   TableHeaderFPDT
	Reserved uint32
	// ResetEnd is the approximate start time of the firmware.
	ResetEnd uint64
	// OSLoaderLoadImageStart is the start time of the boot loader.
	OSLoaderLoadImageStart  uint64
	OSLoaderStartImageStart uint64
	ExitBootServicesEntry   uint64
	// ExitBootServicesExit is the end time of the boot loader.
	ExitBootServicesExit uint64
}

// BootTimeRecord contains the duration of the boot time stages provided by
// the ACPI FPDT table.
type BootTimeRecord struct {
	Firmware time.Duration
	Loader   time.Duration
}

// RetrieveBootTimeRecord attempts to read boot times from Sysfs (Kernel 5.12+)
// and falls back to reading raw ACPI tables via /dev/mem.
func RetrieveBootTimeRecord() (*BootTimeRecord, error) {
	if times, err := retrieveBootTimeWithSysfs(); err == nil {
		return times, nil
	}

	return retrieveBootTimeFromTablePointer() // requires root access
}

// retrieveBootTimeWithSysfs reads parsed values from "/sys/firmware/acpi/fpdt/".
func retrieveBootTimeWithSysfs() (*BootTimeRecord, error) {
	launchNs, err := readParsedSysfsAttribute("bootloader_launch_ns")
	if err != nil {
		return nil, fmt.Errorf("reading attribute bootloader_launch_ns: %w", err)
	}

	exitNs, err := readParsedSysfsAttribute("exitbootservice_end_ns")
	if err != nil {
		return nil, fmt.Errorf("reading attribute exitbootservice_end_ns: %w", err)
	}

	return &BootTimeRecord{
		Firmware: time.Duration(launchNs) * time.Nanosecond,
		Loader:   time.Duration(exitNs-launchNs) * time.Nanosecond,
	}, nil
}

func readParsedSysfsAttribute(attribute string) (uint64, error) {
	path := filepath.Join(pathFPDTBootDir, attribute)
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return 0, fmt.Errorf("reading file %s: %w", path, err)
	}

	d, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing uint: %w", err)
	}

	return d, nil
}

func retrieveBootTimeFromTablePointer() (*BootTimeRecord, error) {
	data, err := os.ReadFile(filepath.Clean(pathFPDTTableFile))
	if err != nil {
		return nil, fmt.Errorf("read FPDT table file %s: %w", pathFPDTTableFile, err)
	}

	if len(data) < tableHeaderSize {
		return nil, errors.New("FPDT table have no header")
	}
	buf := bytes.NewReader(data[tableHeaderSize:]) // skip the header

	var fpdtAddress *uint64

	for buf.Len() > 0 {
		var sh TableHeaderFPDT
		headerBytes := make([]byte, 4)
		if _, err := buf.ReadAt(headerBytes, 0); err != nil {
			break
		}
		binary.Read(bytes.NewReader(headerBytes), binary.LittleEndian, &sh)

		if sh.Length == 0 {
			break // Avoid infinite loop
		}

		recordData := make([]byte, sh.Length)
		if _, err := buf.Read(recordData); err != nil {
			break
		}

		if sh.Type == 0 {
			var ptrRec TablePointerRecordFPDT
			if err := binary.Read(bytes.NewReader(recordData), binary.LittleEndian, &ptrRec); err == nil {
				fpdtAddress = &ptrRec.Address
				break
			}
		}
	}

	if fpdtAddress == nil {
		return nil, errors.New("FPDT pointer not found in FPDT table")
	}

	record, err := readFPDTFromMemory(int64(*fpdtAddress))
	if err != nil {
		return nil, fmt.Errorf("reading FPDT table from address %x: %w", fpdtAddress, err)
	}

	return record, nil
}

func readFPDTFromMemory(physAddr int64) (*BootTimeRecord, error) {
	mem, err := os.Open(filepath.Clean(pathDevMem))
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", pathDevMem, err)
	}
	defer mem.Close()

	headerBuf := make([]byte, tableHeaderSize)
	if _, err := mem.ReadAt(headerBuf, physAddr); err != nil {
		return nil, fmt.Errorf("reading ACPI table header: %w", err)
	}

	var hdr TableHeader
	if err := binary.Read(bytes.NewReader(headerBuf), binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("parsing ACPI table header: %w", err)
	}

	if string(hdr.Signature[:]) != "FPDT" {
		return nil, fmt.Errorf("table signature memory is not FPDT, but %s", hdr.Signature)
	}

	tableData := make([]byte, hdr.Length)
	if _, err := mem.ReadAt(tableData, physAddr); err != nil {
		return nil, fmt.Errorf("reading full table: %w", err)
	}

	offset := tableHeaderSize // skip header
	for offset < int(hdr.Length) {
		r := bytes.NewReader(tableData[offset:])
		var sh TableHeaderFPDT
		if err := binary.Read(r, binary.LittleEndian, &sh); err != nil {
			break
		}

		if sh.Length == 0 {
			break
		}

		if sh.Type == 2 {
			var rec TableRecordFPDT
			r = bytes.NewReader(tableData[offset:]) // reset reader to the record start
			if err := binary.Read(r, binary.LittleEndian, &rec); err != nil {
				return nil, fmt.Errorf("parsing boot record: %w", err)
			}

			result := &BootTimeRecord{}

			// Firmware = Time until Loader Starts
			if rec.OSLoaderLoadImageStart > 0 {
				result.Firmware = time.Duration(rec.OSLoaderLoadImageStart) * time.Nanosecond
			} else if rec.ResetEnd > 0 {
				result.Firmware = time.Duration(rec.ResetEnd) * time.Nanosecond
			}

			// Loader = Time from Loader Start until ExitBootServices (Kernel handover)
			if rec.ExitBootServicesExit > 0 && rec.OSLoaderLoadImageStart > 0 {
				if rec.ExitBootServicesExit > rec.OSLoaderLoadImageStart {
					result.Loader = time.Duration(rec.ExitBootServicesExit-rec.OSLoaderLoadImageStart) * time.Nanosecond
				}
			}

			return result, nil
		}

		offset += int(sh.Length)
	}

	return nil, errors.New("no boot performance record found in FPDT")
}
