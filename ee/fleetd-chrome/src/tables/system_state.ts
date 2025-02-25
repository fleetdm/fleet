import Table from "./Table";

export default class TableSystemState extends Table {
  name = "system_state";
  columns = ["idle_state"];

  async generate() {
    if (!chrome.idle.getAutoLockDelay) {
      return {
        data: [],
        warnings: [
          {
            column: "idle_state",
            error_message: "chrome.idle.getAutoLockDelay API is only available on ChromeOS for screen lock details",
          },
        ],
      };
    }

    // @ts-ignore ignore typing which is out-of-date
    const autoLockDelay: number = await new Promise((resolve) => {
      chrome.idle.getAutoLockDelay((delay) => {
        resolve(delay);
      });
    });

    // Idle time is set to 20% of the user's autolock time or defaults to 30 seconds
    const idleStateDelay = autoLockDelay > 0 ? 0.2 * autoLockDelay : 30;

    // @ts-ignore ignore typing which is out-of-date
    const idleState: "active" | "idle" | "locked" = await new Promise(
      (resolve) => {
        chrome.idle.queryState(idleStateDelay, (state) => {
          resolve(state);
        });
      }
    );

    return { data: [{ idle_state: idleState }] };
  }
}
