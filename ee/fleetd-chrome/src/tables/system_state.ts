import Table from "./Table";

export default class TableSystemState extends Table {
  name = "system_state";
  columns = ["idle_state"];

  async generate() {
    let idle_state;

    try {
      const autoLockDelay = (await new Promise((resolve) =>
        chrome.idle.getAutoLockDelay(resolve)
      )) as number;

      const idleState = await new Promise((resolve) =>
        chrome.idle.queryState(autoLockDelay, resolve)
      );

      idle_state = idleState;
    } catch (err) {
      console.warn("get system state info:", err);
    }

    return [{ idle_state }];
  }
}
