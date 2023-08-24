import Table from "./Table";

export default class TableSystemState extends Table {
  name = "system_state";
  columns = ["idle_state"];

  async generate() {
    // @ts-ignore ignore typing which is out-of-date
    // intentionally don't check for type errors here, as we want them to bubble up to the Fleet layer
    const autoLockDelay: number = await new Promise((resolve) => {
      chrome.idle.getAutoLockDelay((delay) => {
        resolve(delay);
      });
    });

    // Idle time is set to 20% of the user's autolock time or defaults to 30 seconds
    const idleStateDelay = autoLockDelay > 0 ? 0.2 * autoLockDelay : 30;

    // @ts-ignore ignore typing which is out-of-date
    // again, intentionally don't check for type errors here
    const idleState: "active" | "idle" | "locked" = await new Promise(
      (resolve) => {
        chrome.idle.queryState(idleStateDelay, (state) => {
          resolve(state);
        });
      }
    );

    return [{ idle_state: idleState }];
  }
}
