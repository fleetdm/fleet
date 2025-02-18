### Research: extracting name and version from installer packages

> [!WARNING]
> This document is about extracting name and version from the installers, not
> about actually installing them on the device.
>
> For example, extracting info from `.dmg` files is hard for us, but installing
> those files should be a low effort task.

| Type   | Eng effort | Accuracy | UX notes                                    |
| ------ | ---------- | -------- | ------------------------------------------- |
| `.dmg` | High       | Medium   | -                                           |
| `.msi` | Medium     | Medium   | -                                           |
| `.app` | Low        | High     | It's a folder, needs compression to upload. |
| `.pkg` | Low        | High     | -                                           |
| `.exe` | Low        | High     | -                                           |
| `.deb` | Low        | High     | -                                           |
| `.rpm` | Low        | High     | -                                           |

More details:

- Draft PR with a PoC implementation for `.app`, `.exe`, `.pgk`, `.deb` and half of `.msi` in #18232
- Research notes with more details for each type below
- Additional concerns at the end of this doc

### Windows Installer (.msi)

`.msi` files are a relational database (!) laid out in [CFB format](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-cfb/53989ce4-7b05-4f8d-829b-d08d6148375b)

Getting the database tables in binary format form the CBF file is
[possible](https://github.com/fleetdm/fleet/blob/85ee1f7bb9fe33ece20aca0f38678fb5390d3e9c/pkg/file/msi.go#L40-L41), but extracting the information from the tables is a challenge
because the DB format is closed source and Microsoft doesn't disclose any
details about the implementation.

That's why this is labeled as a `Medium` engineering effort.

The strategy to parse the DB files is to rely on two tables: `_StringData` and `_StringPool`,
that contain all unique strings in the DB:

> there is a single stream in the MSI file that holds all the strings. This
> stream is called the string pool contains a single entry for each unique
> string. That way a string column in a table is just an integer offset into
> the string pool.
>
> Source: https://robmensching.com/blog/posts/2003/11/25/inside-the-msi-file-format/

One possibly, but very low accuracy strategy could be to regex the contents of
`_StringData` for anything that looks like an application name or version.

A more sophisticated approach is taken by [this Python library](https://github.com/binref/refinery/blob/de99c87f6dedd6d42508a3d436b6df9181837e34/refinery/units/formats/msi.py#L131) that is able to reverse engineer some of the data based on both tables:

```
$ emit fleet-osquery.msi | ./pyenv/bin/xtmsi MsiTables.json | jq '.Property[] | select(.Property == "ProductName" or .Property == "ProductVersion")'
{
  "Property": "ProductName",
  "Value": "Fleet osquery"
}
{
  "Property": "ProductVersion",
  "Value": "1.22.0"
}
```

A partial implementation that reads the CFB format can be found [here](https://github.com/fleetdm/fleet/blob/85ee1f7bb9fe33ece20aca0f38678fb5390d3e9c/pkg/file/msi.go).

### Apple Disk Image (.dmg)

From Wikipedia:

> A disk image is a compressed copy of the contents of a disk or folder. Disk
> images have .dmg at the end of their names. To see the contents of a disk
> image, you must first open the disk image so it appears on the desktop or in
> a Finder window.

There are two challenges that make `.dmg` files a High engineering effort:

#### Finding the software

A good mental model would be to imagine `.dmg` files as an USB stick that you
plug in a computer: it can contain anything, there are no rules about the kind
of files or the structure of them.

My proposal to fix this problem would be to go for the 80% of the cases and
extract the information from the first `.app` or `.pkg` file we find and fail
if we don't find anything.

#### Accessing the contents on the server

With the strategy to find the software in place, we still need to access the
dmg contents on the server, from Wikipedia:

> Different file systems can be contained inside these disk images, and there
> is also support for creating hybrid optical media images that contain
> multiple file systems. Some of the file systems supported include
> Hierarchical File System (HFS), HFS Plus (HFS+), File Allocation Table (FAT),
> ISO9660, and Universal Disk Format (UDF).

Becuse we can't mount a `dmg` image in the server, and unless we find a
creative way to hack around this, we'll need to implement the logic to in Go.

The only [library I could find](https://github.com/blacktop/go-apfs) is a WIP,
and failed to open Google Chrome and Slack `dmg` files provided in their
websites, but it's a good starting point if we decide to go this route.

### Application Bundle (.app)

[Application Bundles](https://developer.apple.com/library/archive/documentation/CoreFoundation/Conceptual/CFBundles/BundleTypes/BundleTypes.html#//apple_ref/doc/uid/10000123i-CH101-SW5) can be thought as a file directory with a defined structure and file extension
that macOS treats as a single item.

This folder contains all resources necessary for the app to run. As an example,
this is how the Firefox bundle is structured:

```
/Applications $ tree Firefox.app/ -L 2
Firefox.app/
└── Contents
    ├── CodeResources
    ├── Info.plist
    ├── Library
    ├── MacOS
    ├── PkgInfo
    ├── Resources
    ├── _CodeSignature
    └── embedded.provisionprofile
```

The `Info.plist` file is a required file that contains metadata about the app.
We can read the app version and the display name from there.

Because a bundle is a folder, we'll need to ask the IT admin to upload the
bundle compressed (eg: zip, tar).

Here's how different browsers behave when you try to upload an `.app` using a
file input:

- Firefox treats it as a folder, and won't let you select it as a unit (screenshot)
- Safari and Chrome automatically compresses the folder in zip format (screenshot)

A full implementation that reads the name and version from `Info.plist` can be found [here](https://github.com/fleetdm/fleet/blob/85ee1f7bb9fe33ece20aca0f38678fb5390d3e9c/pkg/file/app.go).

### PKG installers (.pkg)

Under the hood, `.pkg` installers are compressed files in `xar` format.

PKG installers are required to have a [Distribution](https://developer.apple.com/library/archive/documentation/DeveloperTools/Reference/DistributionDefinitionRef/Chapters/Distribution_XML_Ref.html) file from which we can extract the name and version.

A full implementation that reads the name and version from the `Distribution` file
can be found [here](https://github.com/fleetdm/fleet/blob/85ee1f7bb9fe33ece20aca0f38678fb5390d3e9c/pkg/file/xar.go).

### Portable Executable (.exe)

The PE format is well documented in [here](https://learn.microsoft.com/en-us/windows/win32/debug/pe-format)

The Go standard library provides a `"debug/pe"` package that we could use as a starting point, but it's not really tailored to our use case.

The file is composed by different sections, and the name and version can be found in the [`.rsrc` section](https://learn.microsoft.com/en-us/windows/win32/debug/pe-format#the-rsrc-section)

For the PoC, I used a Go library that's a bit heavy but does the heavy lifting for us ([link](https://github.com/fleetdm/fleet/blob/85ee1f7bb9fe33ece20aca0f38678fb5390d3e9c/pkg/file/pe.go))

### .deb

Deb files are `ar` archives that contain a `control.tar` archive with
meta-information, including name and version.

Code that extracts the values can be found [here](https://github.com/sassoftware/relic/blob/6c510a666832163a5d02587bda8be970d5e29b8c/lib/signdeb/control.go#L38-L39)

## Additional considerations

### Security

In many cases, we'll have to write custom parsing logic or rely on third party libraries outside of the standard lib.

Keeping that in mind we should take special care and consider any installer as untrusted input, common attacks for Go servers rely on malformed files that make the server OOM or panic.


