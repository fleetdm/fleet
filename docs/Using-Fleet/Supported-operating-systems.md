

# Supported host operating systems

The following operating system versions are supported by the osquery agent. 

| OS      | Supported Version(s)                    |
| :------ | :-------------------------------------  |
| MacOS   | 10.12+                                  |
| Windows | 10                                      |
| Linux   | CentOS 7.1+,  Ubuntu 16.04+             |


## Some notes on compatibility

### Tables
Not all osquery tables are available in every OS. Please check out the [osquery schema](https://osquery.io/schema/5.2.2/) for detailed information. 

If a table is not available for your host, Fleet will generally handle things behind the scenes for you. 

### M1 Macs
The osquery installer generated for MacOS by `fleetctl package` does not include native support for M1 Macs. Some values returned may reflect the information returned by Rosetta rather than the system. For example, CPU will show up as `i486`. 

### Linux
The osquery installer should run on most Linux distributions where glibc is >= 2.2 (there is ongoing work to make osquery work with glibc 2.12+)


<meta name="pageOrderInSection" value="1200">