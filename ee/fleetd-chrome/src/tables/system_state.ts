import Table from "./Table";

export default class TableSystemState extends Table {
  name = "system_state";
  columns = ["idle_state"];

  async generate() {
    const autoLockDelay = (await new Promise((resolve) =>
      chrome.idle.getAutoLockDelay(resolve)
    )) as number;

    // Idle time is set to 20% of the user's autolock time or defaults to 30 seconds
    const idleStateDelay = autoLockDelay > 0 ? 0.2 * autoLockDelay : 30;

    const idleState = (await new Promise((resolve) =>
      chrome.idle.queryState(idleStateDelay, resolve)
    )) as string;

    return [{ idle_state: idleState }];
  }
}
