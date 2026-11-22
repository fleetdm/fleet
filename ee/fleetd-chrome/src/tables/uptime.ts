import Table from "./Table";

export default class TableUptime extends Table {
  name = "uptime";
  columns = ["days", "hours", "minutes", "seconds", "total_seconds"];

  async generate() {
    const { lastRebootTimestamp } = await chrome.storage.local.get("lastRebootTimestamp");
    if (typeof lastRebootTimestamp !== "number" || !Number.isFinite(lastRebootTimestamp) || lastRebootTimestamp <= 0) {
      return { data: [] };
    }

    const totalSeconds = Math.max(0, Math.floor((Date.now() - lastRebootTimestamp) / 1000));
    return {
      data: [{
        days: Math.floor(totalSeconds / 86400).toString(),
        hours: Math.floor((totalSeconds % 86400) / 3600).toString(),
        minutes: Math.floor((totalSeconds % 3600) / 60).toString(),
        seconds: (totalSeconds % 60).toString(),
        total_seconds: totalSeconds.toString(),
      }],
    };
  }
}