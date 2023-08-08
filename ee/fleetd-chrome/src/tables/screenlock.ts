import Table from "./Table";

export default class TableScreenLock extends Table {
  name = "screenlock";
  columns = ["enabled", "grace_period"];

  async generate() {
    // @ts-ignore ignore typing which is out-of-date
    const delay = (await chrome.idle.getAutoLockDelay()) as number;

    // Converts Chrome response to match Osquery's macOS screenlock schema
    const enabled = delay > 0 ? 1 : 0;
    const gracePeriod = delay > 0 ? delay : -1;

    return [{ enabled, grace_period: gracePeriod }];
  }
}
