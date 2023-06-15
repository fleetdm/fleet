import Table from "./Table";

export default class TableScreenLock extends Table {
  name = "screen_lock";
  columns = ["delay"];

  async generate() {
    let delay;

    try {
      const autoLockDelay = await new Promise((resolve) =>
        chrome.idle.getAutoLockDelay(resolve)
      );
      delay = autoLockDelay;
    } catch (err) {
      console.warn("get screen lock info:", err);
    }

    return [{ delay }];
  }
}
