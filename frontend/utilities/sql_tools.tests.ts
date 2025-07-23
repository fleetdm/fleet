import { checkTable } from "./sql_tools";

describe("checkTable", () => {
  // from https://github.com/fleetdm/fleet/issues/26366
  // and  https://github.com/fleetdm/fleet/issues/30109
  const SQL = `
WITH extension_safety_hub_menu_notifications AS (
	SELECT 
		parse_json.key,
		parse_json.fullkey,
		parse_json.path,
		parse_json.value
	FROM (
	    SELECT file.filename, file.path, file.btime FROM file WHERE path LIKE "/Users/%/Library/Application Support/Google/Chrome/%/Preferences" ORDER BY file.btime DESC limit 20
	    ) as chrome_preferences
	JOIN parse_json ON chrome_preferences.path = parse_json.path
	WHERE parse_json.path LIKE "/Users/%/Library/Application Support/Google/Chrome/%/Preferences" AND (
		fullkey IN ("profile/name", "profile/safety_hub_menu_notifications/extensions/isCurrentlyActive", "profile/safety_hub_menu_notifications/extensions/result/timestamp")
		OR fullkey Like "profile/safety_hub_menu_notifications/extensions/result/triggeringExtensions%"
	)
),
extension_details AS (
	SELECT path,
		CASE WHEN key = 'name' THEN value END AS profile_name,
		CASE WHEN key = 'isCurrentlyActive' THEN value END AS notification_active,
		CASE WHEN key GLOB '*[0-9]' THEN value END AS triggering_extension,
		CASE WHEN key = 'timestamp' THEN value END AS timestamp
	FROM extension_safety_hub_menu_notifications
	GROUP BY path, profile_name, notification_active, triggering_extension, timestamp
),
problematic_extensions AS (
	SELECT 
		path,
		MAX(profile_name) OVER (PARTITION by path) AS profile_name,
		MAX(notification_active) AS notification_active,
		MAX(timestamp) AS timestamp,
		triggering_extension
	FROM extension_details
)
SELECT path,
	profile_name,
	notification_active,
	timestamp,
	triggering_extension
FROM problematic_extensions
WHERE triggering_extension IS NOT NULL AND username NOT LIKE '\_%' ESCAPE '\';
`;
  it("should return only real tables by default", () => {
    const { tables, error } = checkTable(SQL);
    expect(error).toBeNull();
    expect(tables).toEqual(["file", "parse_json"]);
  });

  it("should return all tables if 'includeVirtualTables' is set", () => {
    const { tables, error } = checkTable(SQL, true);
    expect(error).toBeNull();
    // Note that json_each is _not_ returned here, as it is a function table.
    expect(tables).toEqual([
      "file",
      "parse_json",
      "chrome_preferences",
      "extension_safety_hub_menu_notifications",
      "extension_details",
      "problematic_extensions",
    ]);
  });

  it("should return an error if SQL is in valid", () => {
    const result = checkTable("SELECTx * FROM users");
    expect(result.error).not.toBeNull();
  });
});
