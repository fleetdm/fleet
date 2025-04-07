## Hello! Welcome to Fleet's osquery tables documentation.

This folder contains additional documentation that we add on top of the existing documentation for osquery to make the documentation of each table more useful for Fleet users.

Fleet's schema tables live in the `tables/` folder. Each osquery table with Fleet overrides has a corresponding YAML file that will override information in the osquery schema documentation.

The existing documentation data lives in the osquery repo at: https://github.com/osquery/osquery-site/tree/source/src/data/osquery_schema_versions.

You can open PRs against a table's YAML file in the `tables/` folder or the osquery schema file. Just note that the data in a table's YAML file overwrites the osquery data whenever there is a conflict.

After adding or modifying the table's YAML file, move to the `website` directory in the project root and run `node ./node_modules/sails/bin/sails run generate-merged-schema` to generate the merged JSON schema.

When adding a new YAML override to Fleet's osquery schema you can use this template:

```yaml
name: # (required) string - The name of the table.
evented: # boolean - whether or not this table is evented. This value may be required depending on the table's source.
description: |- # (required) string - The description for this table. Note: this field supports markdown
	# Add description here
examples: |- # (optional) string - An example query for this table. Note: This field supports markdown
	# Add examples here
notes: |- # (optional) string - Notes about this table. Note: This field supports markdown.
	# Add notes here
platforms: |- # (optional) array - A list of supported platforms for this table (any of: `darwin`, `windows`, `linux`, `chrome`)
	# Add platforms here
columns: # (required) array - An array of columns in this table
  - name: # (required) string - The name of the column
    description: # (required) string - The column's description
    type: # (required) string - the column's data type
    required: # (required) boolean - whether or not this column is required to query this table.
    platforms: # (optional) array - List of supported platforms, used to clarify when a column isn't available on every platform its table supports (any of: `darwin`, `windows`, `linux`, `chrome`)
```

Alternatively, if you want to add documentation about an osquery table for which we don't have a YAML override, you can find the table's page on the [Fleet website](https://fleetdm.com/tables) and click the "edit page" button. Clicking this button will take you to the GitHub web editor with the template pre-filled. After you add information about the table and its columns, you can open a new pull request to add the new YAML file to Fleet's overrides.
