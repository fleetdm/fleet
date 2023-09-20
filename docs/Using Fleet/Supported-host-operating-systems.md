

# Supported host operating systems

Fleet supports the following operating system versions on hosts. 

| OS      | Supported version(s)                    |
| :------ | :-------------------------------------  |
| MacOS   | 12+                                  |
| Windows | 10+                                     |
| Linux   | CentOS 7.1+,  Ubuntu 16.04+             |
| ChromeOS | 112.0.5615.134+                        |

While Fleet may still function partially or fully with OS versions older than those above, Fleet does not actively test against unsupported versions and does not pursue bugs on them. 

## Some notes on compatibility

### Tables
Not all osquery tables are available for every OS. Please check out the [osquery schema](https://fleetdm.com/tables) for detailed information. 

If a table is not available for your host, Fleet will generally handle things behind the scenes for you. 

### M1 Macs
The osquery installer generated for MacOS by `fleetctl package` does not include native support for M1 Macs. Some values returned may reflect the information returned by Rosetta rather than the system. For example, a CPU will show up as `i486`. 

### Linux
The osquery installer will run on Linux distributions where `glibc` is >= 2.2 (there is ongoing work to make osquery work with `glibc` 2.12+).
If you aren't sure what version of `glibc` your distribution is using, [DistroWatch](https://distrowatch.com/) is a great resource. 


<meta name="pageOrderInSection" value="1200">
<meta name="description" value="This page contains information about operating systems that are compatible with the osquery agent.">
<meta name="navSection" value="The basics">
