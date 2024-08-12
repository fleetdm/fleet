### Multiple ABM/VPP tokens

- Figma: https://www.figma.com/design/j2M1heOh8eZD6LcUJks6HE/%239956-Add-multiple-Apple-Business-Manager-and-Volume-Purchasing-Program-connections?node-id=2-130&t=3vPRYgndVgzoD8e4-0
- Issue: https://github.com/fleetdm/fleet/issues/9956

- Constraints/bussines logic:
    - VPP tokens can be assigned to: "All teams", "No team" or a team. Default is "All teams"
    - ABM tokens can be assigned to: "No team" or a team. Default is "No team"
    - Multiple ABM/VPP tokens per team.
    - Same private key can be used for multiple tokens.

### ABM: gameplan overview

The ADE workflow is summarized here

https://github.com/fleetdm/fleet/blob/0a2a48b6d89ab9e428509eb918724c60c245db60/tools/mdm/apple/glossary-and-protocols.md?plain=1#L216-L235

Currently, Fleet manages a single token, and it does the process described there in a cron job every 30 seconds.

To support multiple tokens, we need to:

- On each cron run, do the process for each token that was uploaded.
- For each host ingested via ABM, keep track of the token that was used to ingest it.
- Every time we need to make an ABM call to assign an ADE profile, choose the right token to use for the host.

### VPP: gameplan overview

The VPP token is used to:

- Get a list of available apps for a team
- Assign an app to a host

To support multiple tokens, we need to:

- Add a form to retrieve all tokens available for a team or "no team" this is `(team OR no team) ∪ all teams`.
- To get a list of available apps: loop over each token, get and dedupe the available apps.
- To assign an app to a host: loop over each token until we find the first one with available licenses, use it to assign the app to the host.
- Track the token used to assign the app to the host.


### Database migrations

- Ticket: https://github.com/fleetdm/fleet/issues/21176

#### Migrations

**Tracking team associations for ABM/VPP tokens**

Many-to-many relationship to be tracked in a join table:

```sql
CREATE TABLE mdm_asset_assignments (
    id INT PRIMARY KEY,
    asset_id INT NOT NULL,
    team_id INT NULL, -- NULL if the asset is assigned to "no team" or "all teams"
    target_platform varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL, -- if NULL, asset is not platform specific
    scope ENUM('all_teams', 'team', 'no_team') NOT NULL,
    FOREIGN KEY (asset_id) REFERENCES mdm_config_assets(id),
    FOREIGN KEY (team_id) REFERENCES teams(id),
);
```

**Adding new tokens to mdm_config_assets**


Allow multiple assets with the same name:

```sql
ALTER TABLE mdm_config_assets
DROP INDEX idx_mdm_config_assets_name_deletion_uuid,
DROP COLUMN deletion_uuid;
```

**Tracking ABM token assignments for a host**

```sql
ALTER TABLE host_dep_assignments
ADD COLUMN abm_token_id int unsigned NOT NULL
FOREIGN KEY fk_hda_abm_token_id (abm_token_id) REFERENCES mdm_config_assets(id) ON DELETE SET NULL
```

**Tracking tokens used to assign VPP apps**

```sql
ALTER TABLE host_vpp_software_installs
ADD COLUMN vpp_token_id int unsigned DEFAULT NULL,
FOREIGN KEY host_vpp_software_installs_vpp_token_id (vpp_token_id) REFERENCES mdm_config_assets(id) ON DELETE SET NULL
```

**Migrating current tokens**

For instances already configured:

- The ABM token is "assigned" to the configured default ABM team. We'll need a DB migration that will make it explicit that the existing ABM token belongs to that team, and that existing ADE-enrolled hosts we ADE-enrolled via that token.
- The VPP token should be assigned to "All teams", and any existing app assignments should be tracked as assigned via that token.

#### Queries

1. To insert a new ABM token:

```sql
-- insert into the table
INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES ("abm_token", "value", "checksum")

-- register the relationship to a team
INSERT INTO mdm_asset_assignments (asset_id, team_id, scope) VALUES (1, 1, "team")

-- register the relationship to no team
INSERT INTO mdm_asset_assignments (asset_id, team_id, scope) VALUES (1, NULL, "no_team")
```

2. To insert a new VPP token:

```sql
-- insert into the table
INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES ("vpp_token", "value", "checksum")

-- register the relationship to a team
INSERT INTO mdm_asset_assignments (asset_id, team_id, scope) VALUES (1, 1, "team")

-- register the relationship to no team
INSERT INTO mdm_asset_assignments (asset_id, team_id, scope) VALUES (1, NULL, "no_team")

-- register the relationship to all teams
INSERT INTO mdm_asset_assignments (asset_id, team_id, scope) VALUES (1, NULL, "all_teams")
```

3. To get the available VPP tokens for a team:

```sql
SELECT * FROM mdm_config_assets mca
JOIN mdm_asset_assignments maa
WHERE
    mca.name = "vpp_token" AND
    (mca.team_id = maa.team_id AND maa.scope = "team")
    OR (mca.team_id IS NULL AND maa.scope = "all_teams")
```

4. To get all the ABM tokens:

```sql
SELECT * FROM mdm_config_assets mca
JOIN mdm_asset_assignments maa
WHERE mca.name = "abm_token"
```

