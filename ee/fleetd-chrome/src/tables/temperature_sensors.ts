import Table from "./Table";

export default class TableTemperatureSensors extends Table {
  name = "temperature_sensors";
  columns = ["celcius", "fahrenheit"];

  async generate() {
    let cpuTemperature;

    const cpuInfo = await chrome.system.cpu.getInfo();
    cpuTemperature = cpuInfo.temperatures.map((celcius) => {
      return {
        celcius,
        fahrenheit: 1.8 * ((celcius as unknown) as number) + 32,
      };
    });

    return cpuTemperature;
  }
}
