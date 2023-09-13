import Table from "./Table";

export default class TableSystemInfo extends Table {
  name = "system_info";
  columns = [
    "uuid",
    "hostname",
    "computer_name",
    "hardware_serial",
    "hardware_vendor",
    "hardware_model",
    "cpu_brand",
    "cpu_type",
    "physical_memory",
  ];

  getComputerName(hostname: string, hwSerial: string): string {
    const prefix = "Chromebook";

    if (!!hostname?.length) {
      return hostname;
    }

    if (!!hwSerial?.length) {
      return `${prefix} ${hwSerial}`;
    }

    return prefix;
  }

  async generate() {
    // @ts-expect-error @types/chrome doesn't yet have instanceID.
    const uuid = (await chrome.instanceID.getID()) as string;

    // TODO should it default to UUID or should Fleet handle it somehow?
    let hostname = "";
    try {
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      hostname = (await chrome.enterprise.deviceAttributes.getDeviceHostname()) as string;
    } catch (err) {
      console.warn("get hostname:", err);
    }

    let hwSerial = "";
    try {
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      hwSerial = await chrome.enterprise.deviceAttributes.getDeviceSerialNumber();
    } catch (err) {
      console.warn("get serial number:", err);
    }

    let hwVendor = "",
      hwModel = "";
    try {
      // This throws "Not allowed" error if
      // https://chromeenterprise.google/policies/?policy=EnterpriseHardwarePlatformAPIEnabled is
      // not configured to enabled for the device.
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      const platformInfo = await chrome.enterprise.hardwarePlatform.getHardwarePlatformInfo();
      hwVendor = platformInfo.manufacturer;
      hwModel = platformInfo.model;
    } catch (err) {
      console.warn("get platform info:", err);
    }

    let cpuBrand = "",
      cpuType = "";
    try {
      const cpuInfo = await chrome.system.cpu.getInfo();
      cpuBrand = cpuInfo.modelName;
      cpuType = cpuInfo.archName;
    } catch (err) {
      console.warn("get cpu info:", err);
    }

    let physicalMemory = "";
    try {
      const memoryInfo = await chrome.system.memory.getInfo();
      physicalMemory = memoryInfo.capacity.toString();
    } catch (err) {
      console.warn("get memory info:", err);
    }

    return [
      {
        uuid,
        hostname,
        computer_name: this.getComputerName(hostname, hwSerial),
        hardware_serial: hwSerial,
        hardware_vendor: hwVendor,
        hardware_model: hwModel,
        cpu_brand: cpuBrand,
        cpu_type: cpuType,
        physical_memory: physicalMemory,
      },
    ];
  }
}
