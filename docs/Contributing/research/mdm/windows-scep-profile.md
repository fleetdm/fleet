# SCEP configuration profile on Windows

I used [Windows MDM POC server](https://github.com/fleetdm/fleet/tree/main/tools/mdm/windows/poc-mdm-server) to try to deliver [ClientCertificateInstall CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/clientcertificateinstall-csp) to my Windows host.

Profile I delivered to host:

```xml
<Atomic>
	<CmdID>1</CmdID>
	<Add>
		<CmdID>2</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">node</Format>
			</Meta>
		</Item>
	</Add>
	<Add>
		<CmdID>3</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/RetryCount</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">int</Format>
			</Meta>
			<Data>3</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>4</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/RetryDelay</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">int</Format>
			</Meta>
			<Data>10</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>5</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/KeyUsage</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">int</Format>
			</Meta>
			<Data>160</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>6</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/KeyLength</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">int</Format>
			</Meta>
			<Data>1024</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>7</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/HashAlgorithm</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">chr</Format>
			</Meta>
			<Data>SHA-1</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>8</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/SubjectName</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">chr</Format>
			</Meta>
			<Data>CN=markotestmdm</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>9</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/EKUMapping</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">chr</Format>
			</Meta>
			<Data>1.3.6.1.4.1.311.10.3.12+1.3.6.1.4.1.311.10.3.4+1.3.6.1.4.1.311.20.2.2+1.3.6.1.5.5.7.3.2</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>10</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/ServerURL</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">chr</Format>
			</Meta>
			<Data>https://f7363c86ea41.ngrok-free.app/scep</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>11</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/Challenge</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">chr</Format>
			</Meta>
			<Data>secret</Data>
		</Item>
	</Add>
	<Add>
		<CmdID>12</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/CAThumbprint</LocURI>
			</Target>
			<Meta>
				<Format xmlns="syncml:metinf">chr</Format>
			</Meta>
			<Data>3559DC5D5C4017BD4CDBD0466AF14DDA41655E57</Data>
		</Item>
	</Add>
	<Exec>
		<CmdID>13</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/060358dc442f/Install/Enroll</LocURI>
			</Target>
		</Item>
	</Exec>
</Atomic>
```

## How Windows SCEP profile and SCEP client works

- Windows profile (ClientCertificateInstall CSP) creates an item in the registry (`Computer\HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SCEP\{mdm_server_name}\<ID_FROM_THE_LOC_URI>`)
- In that registry, it stores information that the Windows SCEP client needs to generate a CSR
- I tried using [micromdm/scep](https://github.com/micromdm/scep), but wasn't able to get it work.
  - I think CSP is made to work with NDES server
  - Windows SCEP client apppends `/pkiclient.exe` at the end of the URL added through profile (`ServerURL`)
 
### Screenshots of registry




