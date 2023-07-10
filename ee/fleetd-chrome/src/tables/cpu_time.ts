import Table from "./Table";

export default class TableCpuTime extends Table {
  name = "cpu_time";
  columns = ["idle", "user", "kernel", "total"];

  async generate() {
    const cpuInfo = await chrome.system.cpu.getInfo();
    const cpuProcessors = cpuInfo.processors;

    let rows = [];
    for (let processors of cpuProcessors) {
      // Remove usage key from array returned from API
      rows.push(processors.usage);
    }

    return rows;
  }
}
