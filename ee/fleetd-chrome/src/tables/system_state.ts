import Table from "./Table";

export default class TableSystemState extends Table {
  name = "system_state";
  columns = ["idle_state"];

  async generate() {
    // @ts-ignore ignore typing which is out-of-date
    const autoLockDelay = (await chrome.idle.getAutoLockDelay()) as number;

    // Idle time is set to 20% of the user's autolock time or defaults to 30 seconds
    const idleStateDelay = autoLockDelay > 0 ? 0.2 * autoLockDelay : 30;

    // @ts-ignore ignore typing which is out-of-date
    const idleState = (await chrome.idle.queryState(idleStateDelay)) as string;

    return [{ idle_state: idleState }];
  }
}
