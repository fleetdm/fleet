# Osquery 5.11.0 | VSCode, Apple silicon, and more.

![osquery 5.11.0](../website/assets/images/articles/osquery-5.11.0-cover-1600x900@2x.png)

Osquery 5.11 introduces enhancements that include the `vscode_extensions` table for inventorying VSCode extensions, additional Apple Silicon support columns in the `secureboot` table, Windows shortcut metadata parsing in the `file` table, and caching mechanisms for macOS keychain tables to prevent corruption. Openness is a key Fleet value. We welcome contributions to Fleet and find ways to contribute to other open-source projects. When you support Fleet, you are also contributing to projects like osquery. Let’s take a look at the changes in this latest release.

Please note that osquery 5.11 has already been pushed to Fleet’s stable and edge auto-update channels.


## Highlights



* VSCode extensions table
* Apple silicon support added to `secureboot` table
* Shortcut metadata parsing on Windows
* Preventing keychain corruption with smart caching


### VSCode extensions table

Osquery introduces a new table named `vscode_extensions`, which expands the tool's capabilities in inventory management. This addition allows for the enumeration of extensions installed in Visual Studio Code (VSCode), providing valuable insights into the development environments across managed devices. With this table, IT and security teams can efficiently gather detailed information about VSCode extensions, aiding in compliance checks, security assessments, and the overall management of software assets.


### Apple silicon support added to `secureboot` table

The `secureboot` table in osquery has been updated to include new columns that provide deeper insights into the security configurations of Apple Silicon devices. The added columns are "description," "allow_kernel_extensions," and "allow_mdm_operations," which reflect the settings available in the Startup Security Utility of macOS. This enhancement enables a more detailed analysis of secure boot settings, facilitating better security posture assessments for Apple Silicon devices. This contribution was made by Zach Wasserman, Cofounder of Fleet.


### Shortcut metadata parsing on Windows

The `file` table in osquery has been enhanced to include parsing for shortcut metadata on Windows systems. This update allows for extracting and analyzing information from Windows shortcut files (`.lnk` files), such as target path, arguments, and other relevant shortcut details. This feature provides a more comprehensive understanding of the files present on Windows hosts, aiding in forensic investigations and system audits by offering insights into shortcut configurations and their associated actions. This addition further extends osquery's utility in providing detailed system information and contributes to its role as a valuable tool for IT and security professionals.


### Preventing keychain corruption with smart caching

Caching and throttling mechanisms have been introduced for the `certificates`, `keychain_acls`, and `keychain_items` tables on macOS to mitigate the risk of keychain corruption, a known issue from unstable macOS APIs. The new cache system evaluates if a keychain file has been altered by comparing its SHA256 hash to previous accesses. Should the file remain unchanged or accessed within a preset interval, osquery will reuse the cached results, reducing unnecessary file reads.

This caching operates individually across each table, allowing concurrent yet controlled access to keychain files, enhancing osquery's efficiency and stability on macOS platforms. This significant enhancement was contributed by Fleetie, Victor Lyuboslavsky.


<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-02-16">
<meta name="articleTitle" value="osquery 5.11.0 | VSCode, Apple silicon, and more">
<meta name="articleImageUrl" value="../website/assets/images/articles/osquery-5.11.0-cover-1600x900@2x.png">
