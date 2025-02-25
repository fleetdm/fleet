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
    let warningsArray = [];

    // @ts-expect-error @types/chrome doesn't yet have instanceID.
    const uuid = await chrome.instanceID.getID();
    let devMode = false;
    if (!chrome.enterprise) {
      const { installType } = await chrome.management.getSelf();
      devMode = installType === "development";
    }

    // TODO should it default to UUID or should Fleet handle it somehow?
    let hostname = "";
    try {
      if (!devMode) {
        // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
        hostname = (await chrome.enterprise.deviceAttributes.getDeviceHostname()) as string;
      } else {
        hostname = uuid;
      }
    } catch (err) {
      console.warn("get hostname:", err);
      warningsArray.push({
        column: "hostname",
        error_message: err.message.toString(),
      });
    }

    let hwSerial = "";
    try {
      if (!devMode) {
        // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
        hwSerial = (await chrome.enterprise.deviceAttributes.getDeviceSerialNumber()) as string;
      } else {
        // We leave it blank. The host will be identified by UUID instead.
        hwSerial = "";
      }
    } catch (err) {
      console.warn("get serial number:", err);
      warningsArray.push({
        column: "hardware_serial",
        error_message: err.message.toString(),
      });
    }

    let hwVendor = "",
      hwModel = "";
    try {
      if (!devMode) {
        // This throws "Not allowed" error if
        // https://chromeenterprise.google/policies/?policy=EnterpriseHardwarePlatformAPIEnabled is
        // not configured to enabled for the device.
        // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
        const platformInfo = await chrome.enterprise.hardwarePlatform.getHardwarePlatformInfo();
        hwVendor = platformInfo.manufacturer;
        hwModel = platformInfo.model;
      } else {
        hwVendor = "dev-hardware_vendor";
        hwModel = "dev-hardware_model";
      }
    } catch (err) {
      console.warn("get platform info:", err);
      warningsArray.push({
        column: "hardware_vendor",
        error_message: err.message.toString(),
      });
      warningsArray.push({
        column: "hardware_model",
        error_message: err.message.toString(),
      });
    }

    let cpuBrand = "",
      cpuType = "";
    try {
      const cpuInfo = await chrome.system.cpu.getInfo();
      cpuBrand = cpuInfo.modelName;
      cpuType = cpuInfo.archName;
    } catch (err) {
      console.warn("get cpu info:", err);
      warningsArray.push({
        column: "cpu_brand",
        error_message: err.message.toString(),
      });
      warningsArray.push({
        column: "cpu_type",
        error_message: err.message.toString(),
      });
    }

    let physicalMemory = "";
    try {
      const memoryInfo = await chrome.system.memory.getInfo();
      physicalMemory = memoryInfo.capacity.toString();
    } catch (err) {
      console.warn("get memory info:", err);
      warningsArray.push({
        column: "physical_memory",
        error_message: err.message.toString(),
      });
    }

    return {
      data: [
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
      ],
      warnings: warningsArray,
    };
  }
}
