import Table from "./Table";

export default class TableScreenLock extends Table {
  name = "screenlock";
  columns = ["enabled", "grace_period"];

  async generate() {
    if (!chrome.idle.getAutoLockDelay) {
      return {
        data: [],
        warnings: [
          {
            column: "enabled",
            error_message: "chrome.idle.getAutoLockDelay API is only available on ChromeOS for screen lock details",
          },
        ],
      };
    }
    const delay = await new Promise((resolve) =>
      chrome.idle.getAutoLockDelay(resolve)
    );
    if (typeof delay === "number") {
      // Converts Chrome response to match Osquery's macOS screenlock schema
      const enabled = delay > 0 ? "1" : "0";
      const gracePeriod = delay > 0 ? delay.toString() : "-1";

      return { data: [{ enabled, grace_period: gracePeriod }] };
    }
    throw new Error(
      "Unexpected response from chrome.idle.getAutoLockDelay - expected number"
    );
  }
}
