# Setup Experience

## Updates

### Database

1. Update software_installs table with

    `install_during_setup: boolean` default false

2. New table for setup experience script same schema as scripts today

3. New table for storing setup experience status `setup_experience_status_results`

    | id  | host   | type   | name   | status | execution_id | error  |
    | --- | ------ | ------ | ------ | ------ | ----------- | ------ |
    | int | string | string | string | string | string     | string |

* type: `bootstrap-package`, `software-install`, `post-install-script`
* status: `pending`, `installing`, `installed`, `failed`, `ran`, `running`

Populate this table with all items as pending when hit /start (remove old matches for this host)

`IDEA 1`: add callback id to software, bootstrap pkg, script to update status. Where callback would
probably be `{"table": "setup_experience_status_results", "id": 1234}`

`IDEA 2`: Handle looking up if this item is 'installed during setup' and see if there is a
corresponding pending item for this host after every run.


### API

* Filter software on macOS custom package only
  * we need to be able to list software available for setup experience which will not include VPP today.
  * Needs to include the new property `installed_during_setup` to indicate which items are currently selected.
* PUT /setup_experience/software
  - Passing an array of software IDs that we want to be enabled, all others are disabled. 

1. (POST GET DELETE) /setup_experience/script

2. GET /:device_token/setup_experience/status
   - {'software': [{'id': 1, 'status': 'installing'}], 'script': {'id': 2, 'status': 'waiting'}}

3. POST /:device_token/setup_experience/start
   - Starts software install and script process on device. 


### Backend MGMT

* Enqueue all software installs on start
* Monitor when all software is finished (success or failed) and enqueue script.
* Release device normally if they don't have any setup software or setup script