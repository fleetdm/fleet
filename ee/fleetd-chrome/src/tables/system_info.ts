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
    // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
    const hostname = (await chrome.enterprise.deviceAttributes.getDeviceHostname()) as string;

    // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
    const hwSerial = (await chrome.enterprise.deviceAttributes.getDeviceSerialNumber()) as string;

    // This throws "Not allowed" error if
    // https://chromeenterprise.google/policies/?policy=EnterpriseHardwarePlatformAPIEnabled is
    // not configured to enabled for the device.
    // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
    const platformInfo = await chrome.enterprise.hardwarePlatform.getHardwarePlatformInfo();
    const hwVendor = platformInfo.manufacturer;
    const hwModel = platformInfo.model;

    const cpuInfo = await chrome.system.cpu.getInfo();
    const cpuBrand = cpuInfo.modelName;
    const cpuType = cpuInfo.archName;

    const memoryInfo = await chrome.system.memory.getInfo();
    const physicalMemory = memoryInfo.capacity.toString();

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
