
# Software Manage Lifecycle

## Add Software to empty db

```mermaid
mindmap
  Software Actions
    Software Upload
      Extract Version and match Title
      Store Query for precondition
      Store Script for Install
      Store Script for post-install
      Store Binary FS / S3
      Storage Driver Configured FS
      Storage Driver Configured S3
    Software exists
      API changes to show on list software
    Delete Software
      Remove Binary
      Clear DB refs for scripts / query
    Install Software to Host
      Run Pre-condition
      Download Binary - API?
      Run install script
      Run Post install script
      Cleanup binary
```

## DB

Notes:

- The diagram below intends to cover everything except the orchestration part.
- Idea:
  - We don't need to report/filter, etc based on orchestration steps, so we don't need to track anything other than failed/success status in the db.
  - As such, all the pre/post/rollback logic can be handled entirely by orbit

```mermaid
---
title: DB schema
---
erDiagram
  software_installers ||--|| software : "software_version"
  software_installers ||--|| script_contents : "install_script"
  software_installers ||--|| script_contents : "post_install_script"
  software_installers {
    int(10) id PK
    bigint(20) software_version FK
    int(10) install_script FK
    int(10) post_install_script FK
    text pre_install_condition
  }

  host_software_installs ||--|| hosts : "host_id"
  host_software_installs ||--|| software_installers : "software_installer_id"
  host_software_installs {
    int(10) id PK
    int(10) host_id FK
    int(10) software_installer_id FK
    text pre_install_condition_output
    varchar(20) pre_install_condition_status FK "references `host_mdm_status`"

    unique_key host_id_software_installer_id
  }

  script_contents {
    int(10) id PK
  }

  software ||--|| software_titles : "title_id"
  software {
    bigint(20) id PK 
    int(10) title_id FK
    varchar(255) version
    varchar(255) name
  }

  software_titles {
    int(10) id PK
    varchar(255) name
  }

  hosts {
    int(10) id PK
  }
```
