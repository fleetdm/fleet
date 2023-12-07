import Table from "./Table";

export default class TableChromeExtensions extends Table {
  name = "chrome_extensions";
  columns = [
    "browser_type",
    "name",
    "identifier",
    "version",
    "description",
    "update_url",
    "permissions",
    "permissions_json",
    "state",
    "path",
  ];

  async generate() {
    const extensions = await chrome.management.getAll();
    let rows = [];
    for (let ext of extensions) {
      // Osquery returns these two permission types merged together which doesn't necessarily seem
      // like the most intuitive option, but we do that to match.
      const mergedPerms = [...ext.permissions, ...ext.hostPermissions];
      rows.push({
        browser_type: "chrome",
        name: ext.name,
        identifier: ext.id,
        version: ext.version,
        description: ext.description,
        update_url: ext.updateUrl,
        permissions: mergedPerms.join(", "),
        permissions_json: JSON.stringify(mergedPerms),
        state: ext.enabled ? "1" : "0",
        path: "",
      });
    }

    return { data: rows };
  }
}
