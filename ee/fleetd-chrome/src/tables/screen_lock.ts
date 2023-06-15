import Table from "./Table";

export default class TableScreenLock extends Table {
  name = "screen_lock";
  columns = ["delay"];

  async generate() {
    let delay;

    try {
      // @ts-ignore
      const delay = await chrome.idle.getAutoLockDelay();
      return [{ delay }];
    } catch (err) {
      console.warn("get screen lock delay info:", err);
    }

    return [{ delay }];
  }
}
