import Table from "./Table";

export default class TableCpuInfo extends Table {
  name = "cpu_info";
  columns = ["model", "logical_processors"];

  async generate() {
    const cpuInfo = await chrome.system.cpu.getInfo();
    const cpuBrand = cpuInfo.modelName;
    const logicalProcessors = cpuInfo.numOfProcessors;

    return [
      {
        model: cpuBrand,
        logical_processors: logicalProcessors,
      },
    ];
  }
}
