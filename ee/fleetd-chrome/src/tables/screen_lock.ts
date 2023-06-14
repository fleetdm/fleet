import Table from "./Table";

export default class TableScreenLock extends Table {
  name = "screen_lock";
  columns = ["delay"];

  async generate() {
    let delay;

    // This uses an old function callback so need to convert the callback style to a promise
    // Consider updating the version so we don't have to use a function callback
    // Use ts ignore with await and get rid of the promise
    // try {
    //   const delay = await new Promise((resolve) =>
    //     chrome.idle.getAutoLockDelay(resolve)
    //   );
    //   return [{ delay }];
    // } catch (err) {
    //   console.warn("get screen lock info:", err);
    // }

    try {
      // @ts-ignore
      const delay = await chrome.idle.getAutoLockDelay();
      return [{ delay }];
    } catch (err) {
      console.warn("get memory info:", err);
    }

    return [{ delay }];
  }
}
