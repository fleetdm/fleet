package lvm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// sample from real LUKS encrypted Ubuntu disk
var testJsonUbuntu = `{
   "blockdevices": [
      {
         "name": "loop0",
         "maj:min": "7:0",
         "rm": false,
         "size": "4K",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/bare/5"
         ]
      },{
         "name": "loop1",
         "maj:min": "7:1",
         "rm": false,
         "size": "74.3M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/core22/1564"
         ]
      },{
         "name": "loop2",
         "maj:min": "7:2",
         "rm": false,
         "size": "73.9M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/core22/1663"
         ]
      },{
         "name": "loop3",
         "maj:min": "7:3",
         "rm": false,
         "size": "269.8M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/firefox/4793"
         ]
      },{
         "name": "loop4",
         "maj:min": "7:4",
         "rm": false,
         "size": "10.7M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/firmware-updater/127"
         ]
      },{
         "name": "loop5",
         "maj:min": "7:5",
         "rm": false,
         "size": "11.1M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/firmware-updater/147"
         ]
      },{
         "name": "loop6",
         "maj:min": "7:6",
         "rm": false,
         "size": "505.1M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/gnome-42-2204/176"
         ]
      },{
         "name": "loop7",
         "maj:min": "7:7",
         "rm": false,
         "size": "91.7M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/gtk-common-themes/1535"
         ]
      },{
         "name": "loop8",
         "maj:min": "7:8",
         "rm": false,
         "size": "10.7M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/snap-store/1218"
         ]
      },{
         "name": "loop9",
         "maj:min": "7:9",
         "rm": false,
         "size": "10.5M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/snap-store/1173"
         ]
      },{
         "name": "loop10",
         "maj:min": "7:10",
         "rm": false,
         "size": "38.8M",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/snapd/21759"
         ]
      },{
         "name": "loop11",
         "maj:min": "7:11",
         "rm": false,
         "size": "500K",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/snapd-desktop-integration/178"
         ]
      },{
         "name": "loop12",
         "maj:min": "7:12",
         "rm": false,
         "size": "568K",
         "ro": true,
         "type": "loop",
         "mountpoints": [
             "/snap/snapd-desktop-integration/253"
         ]
      },{
         "name": "nvme0n1",
         "maj:min": "259:0",
         "rm": false,
         "size": "476.9G",
         "ro": false,
         "type": "disk",
         "mountpoints": [
             null
         ],
         "children": [
            {
               "name": "nvme0n1p1",
               "maj:min": "259:1",
               "rm": false,
               "size": "1G",
               "ro": false,
               "type": "part",
               "mountpoints": [
                   "/boot/efi"
               ]
            },{
               "name": "nvme0n1p2",
               "maj:min": "259:2",
               "rm": false,
               "size": "2G",
               "ro": false,
               "type": "part",
               "mountpoints": [
                   "/boot"
               ]
            },{
               "name": "nvme0n1p3",
               "maj:min": "259:3",
               "rm": false,
               "size": "473.9G",
               "ro": false,
               "type": "part",
               "mountpoints": [
                   null
               ],
               "children": [
                  {
                     "name": "dm_crypt-0",
                     "maj:min": "252:0",
                     "rm": false,
                     "size": "473.9G",
                     "ro": false,
                     "type": "crypt",
                     "mountpoints": [
                         null
                     ],
                     "children": [
                        {
                           "name": "ubuntu--vg-ubuntu--lv",
                           "maj:min": "252:1",
                           "rm": false,
                           "size": "473.9G",
                           "ro": false,
                           "type": "lvm",
                           "mountpoints": [
                               "/"
                           ]
                        }
                     ]
                  }
               ]
            }
         ]
      }
   ]
}`

var testJsonFedora = `{
   "blockdevices": [
      {
         "name": "sr0",
         "maj:min": "11:0",
         "rm": true,
         "size": "2.1G",
         "ro": false,
         "type": "rom",
         "mountpoints": [
             "/run/media/luk/Fedora-WS-Live-40-1-14"
         ]
      },{
         "name": "zram0",
         "maj:min": "252:0",
         "rm": false,
         "size": "1.9G",
         "ro": false,
         "type": "disk",
         "mountpoints": [
             "[SWAP]"
         ]
      },{
         "name": "nvme0n1",
         "maj:min": "259:0",
         "rm": false,
         "size": "20G",
         "ro": false,
         "type": "disk",
         "mountpoints": [
             null
         ],
         "children": [
            {
               "name": "nvme0n1p1",
               "maj:min": "259:1",
               "rm": false,
               "size": "600M",
               "ro": false,
               "type": "part",
               "mountpoints": [
                   "/boot/efi"
               ]
         },{
               "name": "nvme0n1p2",
               "maj:min": "259:2",
               "rm": false,
               "size": "1G",
               "ro": false,
               "type": "part",
               "mountpoints": [
                   "/boot"
               ]
            },{
               "name": "nvme0n1p3",
               "maj:min": "259:3",
               "rm": false,
               "size": "18.4G",
               "ro": false,
               "type": "part",
               "mountpoints": [
                   null
               ],
               "children": [
                  {
                     "name": "luks-21fc9b67-752e-42fb-83bb-8c92864382e9",
                     "maj:min": "253:0",
                     "rm": false,
                     "size": "18.4G",
                     "ro": false,
                     "type": "crypt",
                     "mountpoints": [
                         "/home", "/"
                     ]
                  }
               ]
            }
         ]
      }
   ]
}`

var testJsonOther = `{
    "blockdevices": [
        {
            "name": "loop0",
            "maj:min": "7:0",
            "rm": false,
            "size": "4K",
            "ro": true,
            "type": "loop",
            "mountpoint": "/snap/bare/5"
        },
        {
            "name": "loop1",
            "maj:min": "7:1",
            "rm": false,
            "size": "346.3M",
            "ro": true,
            "type": "loop",
            "mountpoint": "/snap/gnome-3-38-2004/119"
        },
        {
            "name": "loop2",
            "maj:min": "7:2",
            "rm": false,
            "size": "49.9M",
            "ro": true,
            "type": "loop",
            "mountpoint": "/snap/snapd/18357"
        },
        {
            "name": "loop3",
            "maj:min": "7:3",
            "rm": false,
            "size": "46M",
            "ro": true,
            "type": "loop",
            "mountpoint": "/snap/snap-store/638"
        },
        {
            "name": "loop4",
            "maj:min": "7:4",
            "rm": false,
            "size": "63.3M",
            "ro": true,
            "type": "loop",
            "mountpoint": "/snap/core20/1828"
        },
        {
            "name": "loop5",
            "maj:min": "7:5",
            "rm": false,
            "size": "91.7M",
            "ro": true,
            "type": "loop",
            "mountpoint": "/snap/gtk-common-themes/1535"
        },
        {
            "name": "nvme0n1",
            "maj:min": "259:0",
            "rm": false,
            "size": "953.9G",
            "ro": false,
            "type": "disk",
            "mountpoint": null,
            "children": [
                {
                    "name": "nvme0n1p1",
                    "maj:min": "259:1",
                    "rm": false,
                    "size": "512M",
                    "ro": false,
                    "type": "part",
                    "mountpoint": "/boot/efi"
                },
                {
                    "name": "nvme0n1p2",
                    "maj:min": "259:2",
                    "rm": false,
                    "size": "1.4G",
                    "ro": false,
                    "type": "part",
                    "mountpoint": "/boot"
                },
                {
                    "name": "nvme0n1p3",
                    "maj:min": "259:3",
                    "rm": false,
                    "size": "952G",
                    "ro": false,
                    "type": "part",
                    "mountpoint": null,
                    "children": [
                        {
                            "name": "nvme0n1p3_crypt",
                            "maj:min": "253:0",
                            "rm": false,
                            "size": "951.9G",
                            "ro": false,
                            "type": "crypt",
                            "mountpoint": null,
                            "children": [
                                {
                                    "name": "vgubuntu-root",
                                    "maj:min": "253:1",
                                    "rm": false,
                                    "size": "930.4G",
                                    "ro": false,
                                    "type": "lvm",
                                    "mountpoint": "/"
                                },
                                {
                                    "name": "vgubuntu-swap_1",
                                    "maj:min": "253:2",
                                    "rm": false,
                                    "size": "976M",
                                    "ro": false,
                                    "type": "lvm",
                                    "mountpoint": "[SWAP]"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    ]
}`

func TestFindRootDisk(t *testing.T) {
	var input bytes.Buffer
	_, err := input.WriteString(testJsonUbuntu)
	assert.NoError(t, err)

	output, err := rootDiskFromJson(input)
	assert.NoError(t, err)
	assert.Equal(t, "/dev/nvme0n1p3", output)

	input = bytes.Buffer{}
	_, err = input.WriteString(testJsonFedora)
	assert.NoError(t, err)

	output, err = rootDiskFromJson(input)
	assert.NoError(t, err)
	assert.Equal(t, "/dev/nvme0n1p3", output)

	input = bytes.Buffer{}
	_, err = input.WriteString(testJsonOther)
	assert.NoError(t, err)

	output, err = rootDiskFromJson(input)
	assert.NoError(t, err)
	assert.Equal(t, "/dev/nvme0n1p3", output)
}

func TestErrorNoMountPoint(t *testing.T) {
	var input bytes.Buffer
	_, err := input.WriteString(`{"blockdevices": [{"name": "nvme0n1", "mountpoints": [null]}]}`)
	assert.NoError(t, err)

	output, err := rootDiskFromJson(input)
	assert.Error(t, err)
	assert.Empty(t, output)
}

func TestErrorNoRootPartition(t *testing.T) {
	var input bytes.Buffer
	_, err := input.WriteString(`{"blockdevices": [{"name": "nvme0n1", "mountpoints": ["/boot"]}]}`)
	assert.NoError(t, err)

	output, err := rootDiskFromJson(input)
	assert.Error(t, err)
	assert.Empty(t, output)
}

func TestErrorInvalidJson(t *testing.T) {
	var input bytes.Buffer
	_, err := input.WriteString(`{`)
	assert.NoError(t, err)

	output, err := rootDiskFromJson(input)
	assert.Error(t, err)
	assert.Empty(t, output)
}
