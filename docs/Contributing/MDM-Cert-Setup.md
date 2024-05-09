# MDM Cert Setup

At the end of this process you will have 8 files, 1 only needed temporarily and will be using them
to set 10 environment variables to enable both Apple and Windows MDM capabilities.

```mermaid
flowchart LR
  st["Click: Request Push Cert"]
  sc["SCEP Cert: fleet-mdm-apple-scep.crt"]
  sk["SCEP Key: fleet-mdm-apple-scep.key"]
  pem["APNS Cert: MDM_ Fleet Device Management Inc_Certificate.pem"]
  apnsk["APNS Key: mdmcert.download.push.key"]
  csr["apple-apn-csr.txt"]
  apns_portal["APNS Portal"]
  st -- Email --> csr
  st -- DL Browser --> sc
  st -- DL Browser --> sk
  st -- DL Browser --> apnsk
  csr -- Create new certificate upload -->apns_portal
  apns_portal -- Download -->pem
```

> Rename .pem file to mdmcert.download.push.pem (for your sanity to know it's linked to the APNS Key
> you got in the first step)

Relaunch Fleet with these variables configured (you can change scepchallenge to another word or
phrase if you'd like)
```
FLEET_MDM_APPLE_SCEP_CHALLENGE: scepchallenge
FLEET_MDM_APPLE_APNS_CERT: path_to/mdmcert.download.push.pem
FLEET_MDM_APPLE_APNS_KEY: path_to/mdmcert.download.push.key
FLEET_MDM_APPLE_SCEP_CERT: path_to/fleet-mdm-apple-scep.crt
FLEET_MDM_APPLE_SCEP_KEY: path_to/fleet-mdm-apple-scep.key
# You are allowed to reuse scep cert/key for Windows
FLEET_MDM_WINDOWS_WSTEP_IDENTITY_CERT: path_to/fleet-mdm-apple-scep.crt
FLEET_MDM_WINDOWS_WSTEP_IDENTITY_KEY: path_to/fleet-mdm-apple-scep.key
```

```mermaid
flowchart LR
  st["ABM Section Click Download"]
  abm_cert["ABM Cert: fleet-apple-mdm-bm-public.crt"]
  abm_key["ABM Key: fleet-apple-mdm-bm-private.key"]
  abm_server["ABM Server - create new"]
  dltoken["ABM Token: ServerName_Token_Datetime_smime.p7m"]
  st --> abm_cert
  st --> abm_key
  abm_cert -- Create Server and upload cert --> abm_server
  abm_server -- Download token --> dltoken
```

> Rename .p7m file to downloadtoken.p7m (again for your sanity)
Restart Server with new env vars

```
FLEET_MDM_APPLE_BM_SERVER_TOKEN: path_to/downloadtoken.p7m
FLEET_MDM_APPLE_BM_CERT: path_to/fleet-apple-mdm-bm-public-key.crt
FLEET_MDM_APPLE_BM_KEY: path_to/fleet-apple-mdm-bm-private.key
```

### Start the process

* Navigate to the Settings > Integrations > Mobile device management (MDM) page.
* Under Apple Push Certificates Portal, select Request, then fill out the form. This should generate
  three files and send an email to you with an attached CSR file.

...

## Theoretical Revision

Navigate to Settings > Integrations > MDM
```mermaid
flowchart LR
  st["Click Enable MDM"]
  prompt["Prompt for email"]
  csr["Download CSR"]
  Server["MySQL DB"]
  st -- Error if current user email is a blocked email --> prompt
  st -- If current user email not blocked -->csr
  st -- Store apns key --> Server
  st -- Store SCEP key --> Server
  st -- Store SCEP cert --> Server
  st -- Store csr.txt? --> Server
```

```mermaid
flowchart LR
  csr["csr.txt"]
  pp["APNS Portal"]
  pem["APN Cert.pem"]
  ff["Fleet server"]
  db["MySQL DB"]
  csr -- Create New Cert --> pp
  pp -- Download --> pem
  pem -- Upload Form --> ff
  ff -- Store --> db
```

Fleet should now be able to dynamically check all certs and "enable MDM" as if we restarted w/ the
env vars set

```mermaid
flowchart LR
  fl_abm["Fleet ABM Section Download"]
  bm_cert["bm_public.crt"]
  db["MySQL DB"]
  token
  ABM
  fl_abm -- Store ABM Key --> db
  fl_abm -- Store ABM Cert --> db
  fl_abm -- Download Cert --> bm_cert
  bm_cert -- Create Server --> ABM
  ABM -- Download token --> token
  token -- Upload form and store in db enable abm --> fl_abm
```

new flow would:
* click download csr and open modal waiting for file upload of apns cert with a link to portal
* User in new tab create cert in apns portal create / download
* return to modal and upload apns cert
* Fleet dynamically reloads MDM (ABM section shows now)
* Click download abm cert opens modal waiting for download token with link to abm
* Create server with cert and download token
* return to modal and upload token and save

DONE