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

  async generate() {
    // @ts-expect-error @types/chrome doesn't yet have instanceID.
    const uuid = await chrome.instanceID.getID();

    // TODO should it default to UUID or should Fleet handle it somehow?
    let hostname = "";
    try {
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      hostname = (await chrome.enterprise.deviceAttributes.getDeviceHostname()) as string;
    } catch (err) {
      console.warn("get hostname:", err);
    }

    let hardware_serial = "";
    try {
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      hardware_serial = await chrome.enterprise.deviceAttributes.getDeviceSerialNumber();
    } catch (err) {
      console.warn("get serial number:", err);
    }

    let hardware_vendor = "",
      hardware_model = "";
    try {
      // This throws "Not allowed" error if
      // https://chromeenterprise.google/policies/?policy=EnterpriseHardwarePlatformAPIEnabled is
      // not configured to enabled for the device.
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      const platform_info = await chrome.enterprise.hardwarePlatform.getHardwarePlatformInfo();
      hardware_vendor = platform_info.manufacturer;
      hardware_model = platform_info.model;
    } catch (err) {
      console.warn("get platform info:", err);
    }

    let cpu_brand = "",
      cpu_type = "";
    try {
      const cpu_info = await chrome.system.cpu.getInfo();
      cpu_brand = cpu_info.modelName;
      cpu_type = cpu_info.archName;
    } catch (err) {
      console.warn("get cpu info:", err);
    }

    let physical_memory = "";
    try {
      const memory_info = await chrome.system.memory.getInfo();
      physical_memory = memory_info.capacity.toString();
    } catch (err) {
      console.warn("get memory info:", err);
    }

    return [
      {
        uuid,
        hostname,
        computer_name: hostname ? hostname : `ChromeOS ${hardware_serial}`,
        hardware_serial,
        hardware_vendor,
        hardware_model,
        cpu_brand,
        cpu_type,
        physical_memory,
      },
    ];
  }
}
