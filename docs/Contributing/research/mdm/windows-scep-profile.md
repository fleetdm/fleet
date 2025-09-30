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
  - Windows SCEP client appends`/pkiclient.exe` at the end of the URL added through profile (`ServerURL`)
- If you craft a profile correctly, it will add an item to the registry, and the MDM response from the host will return 200, meaning SCEP information is added to registry, but that doesn't mean that the SCEP request will be successful. In screenshots below, see how my profile returned `200`, but the SCEP request to the server failed.
- After SCEP information is added to the registry, Windows creates a scheduled task that runs the SCEP client (a few times if it fails at first).
- It seems that ClientCertificateInstall expects NDES server, so it might be that other SCEP servers might need to adjust to accept requests from Windows client. Maybe URL rewrite could work?
  - I noticed that people from smallstep joined the conversation. There might be some useful information here: https://github.com/micromdm/scep/issues/238
 
### Screenshots of registry

Here is all the SCEP information added to a registry, with the subdirectory named after the unique ID from the SCEP profile.
<img width="1422" height="808" alt="win-registry-1" src="https://github.com/user-attachments/assets/242cf41a-6add-4be2-a1a8-0850341935a1" />

One level up, there is an `ErrorCode` field if the SCEP request fails.
<img width="1434" height="821" alt="win-registry-2" src="https://github.com/user-attachments/assets/db029a6d-78d3-4991-8deb-8845f4e99254" />

In Event Viewer, I was able to find SCEP request failure, becaues Windows client appended `/pkiclient.exe` at the end of the server URL.
<img width="1920" height="1129" alt="win-event-viewer" src="https://github.com/user-attachments/assets/fe5880b7-8458-41d7-ac24-2717e11ad9c0" />
