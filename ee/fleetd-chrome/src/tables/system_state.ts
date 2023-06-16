import Table from "./Table";

export default class TableSystemState extends Table {
  name = "system_state";
  columns = ["idle_state"];

  async generate() {
    const autoLockDelay = (await new Promise((resolve) =>
      chrome.idle.getAutoLockDelay(resolve)
    )) as number;

    const idleState = (await new Promise((resolve) =>
      chrome.idle.queryState(autoLockDelay, resolve)
    )) as string;

    return [{ idle_state: idleState }];
  }
}
