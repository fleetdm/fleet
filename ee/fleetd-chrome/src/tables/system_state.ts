import Table from "./Table";

export default class TableSystemState extends Table {
  name = "system_state";
  columns = ["idle_state"];

  async generate() {
    let idle_state;

    try {
      // @ts-ignore
      const delay = await chrome.idle.getAutoLockDelay();
      // @ts-ignore
      const idle_state = await chrome.idle.queryState(delay);

      return [{ idle_state }];
    } catch (err) {
      console.warn("get system state info:", err);
    }

    return [{ idle_state }];
  }
}
