# Fleet Runway (Startup experience)

## Workflow

1. Admin adds N software
2. Admin adds 1 script
3. DEP holds release and adds install fleet desktop and swift dialog before release.
4. swift dialog launches asap after release and is says `fleet is installing software.
5. All software must be attempted to install (success or failure).
6. Run script success or failure


### Development required for both options

1. Holding release until fleet desktop is installed and running and swift dialog is downloaded and
   ready to run
2. launch swift dialog asap after release and hold until all setup items are done. (button disabled
   until finished then enabled.)
3. API GET list runway software
4. API POST update list runway software
5. API GET runway script
6. API update POST add script to include `runway: true`
7. API PATCH select runway script ID


### Development Option 1 (Policies)

Policy option 1 (file existence)
* Bloat what we are already doing w/ migration and add more files that signal to fleet if setup software needs to install.
  * Policy becomes `if {filepath/filename} exists`
  * Potential problems
    * All current hosts that didn't go through this process will also fail this policy and have software queued.
    * Can't take the upcoming activities / past activities to be the full list of setup experience
      tasks since the script will have to be queued after software has finished.


Policy option 2 (Software existence)
* Initial thoughts would be difficulty in managing when to create / delete the policy
   * Policy becomes `if {app} not in installed apps` (we have heard this may be a bit flaky from
     eops development of the feature this week.)
   * This policy would still install on all hosts not just new ones.

Policy work
* Development creating the policies that would map to the runway
* Making sure when the software or script is removed or changed from the list to remove the
  associated policy.
* prevent admins from deleting the policy assiciated w/ runway.
* Modify Upcoming / past activities to be able to filter for runway items to validate status and
  release.
* OR create a new list endpoint like below that is backed by activities but will be for a specific
  subset of activities.
* Holding the swift dialog while policies first run and potential activities are enqueued even
  though it will look `done` from the start until everything is added in.


### Development Option 2 (Simple execution)

1. API PUT device token to 'start' which enqueues all software.
2. API (optional) GET device token list runway status
    `{'software': [{'id': 1, 'status': 'installing'}], 'script': {'id': 2, 'status': 'waiting'}}`
3. Either server tracks installs and then queue's script OR API PUT device token for finished
   software and ready to queue script
