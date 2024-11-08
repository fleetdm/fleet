package lvm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

type BlockDevice struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Mountpoints []string      `json:"mountpoints"`
	Children    []BlockDevice `json:"children,omitempty"`
}

// FindRootDisk finds the physical partition that
// contains the root filesystem mounted at "/" on a
// LVM Linux volume.
func FindRootDisk() (string, error) {
	cmd := exec.Command("lsblk", "--json")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run lsblk: %w", err)
	}

	return rootDiskFromJson(out)
}

func rootDiskFromJson(input bytes.Buffer) (string, error) {
	var data struct {
		Blockdevices []BlockDevice `json:"blockdevices"`
	}

	if err := json.Unmarshal(input.Bytes(), &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Find the root partition mounted at "/"
	rootPartition := findRootPartition(data.Blockdevices)
	if rootPartition == nil {
		return "", errors.New("root partition not found")
	}

	// Trace up to the nearest parent partition of type "part" (partition)
	physicalPartition := findParentPartitionOfTypePart(data.Blockdevices, rootPartition)
	if physicalPartition == nil {
		return "", errors.New("physical partition of type 'part' not found")
	}

	return fmt.Sprintf("/dev/%s", physicalPartition.Name), nil
}

// findRootPartition recursively searches for the partition
// mounted at "/" within the device tree.
func findRootPartition(devices []BlockDevice) *BlockDevice {
	for _, device := range devices {
		if result := searchForRoot(device); result != nil {
			return result
		}
	}
	return nil
}

// searchForRoot recursively checks each device and its children
// to find the one mounted at "/".
func searchForRoot(device BlockDevice) *BlockDevice {
	for _, mountpoint := range device.Mountpoints {
		if mountpoint == "/" {
			return &device
		}
	}
	for _, child := range device.Children {
		if result := searchForRoot(child); result != nil {
			return result
		}
	}
	return nil
}

// findParentPartitionOfTypePart traverses upwards from the given device to find the nearest "part" type parent.
func findParentPartitionOfTypePart(devices []BlockDevice, target *BlockDevice) *BlockDevice {
	var queue []BlockDevice
	queue = append(queue, devices...)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, child := range current.Children {
			if child.Name == target.Name {
				if current.Type == "part" {
					return &current
				}
				return findParentPartitionOfTypePart(devices, &current)
			}
			queue = append(queue, child)
		}
	}
	return nil
}
