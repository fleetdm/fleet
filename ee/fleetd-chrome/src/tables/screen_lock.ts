import Table from "./Table";

export default class TableScreenLock extends Table {
  name = "screen_lock";
  columns = ["delay"];

  async generate() {
    const delay = (await new Promise((resolve) =>
      chrome.idle.getAutoLockDelay(resolve)
    )) as number;

    return [{ delay }];
  }
}
