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
  ];

  async generate() {
    // @ts-expect-error @types/chrome doesn't yet have instanceID.
    const uuid = await chrome.instanceID.getID();

    // TODO should it default to UUID or should Fleet handle it somehow?
    let hostname = uuid;
    try {
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      hostname = await chrome.enterprise.deviceAttributes.getDeviceHostname();
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
      // TODO figure out why this throws "Not allowed" error on test device
      // @ts-expect-error @types/chrome doesn't yet have the deviceAttributes Promise API.
      const platform_info = await chrome.enterprise.hardwarePlatform.getHardwarePlatformInfo();
      hardware_vendor = platform_info.manufacturer;
      hardware_model = platform_info.model;
    } catch (err) {
      console.warn("get platform info:", err);
    }

    return [
      {
        uuid,
        hostname,
        computer_name: hostname,
        hardware_serial,
        hardware_vendor,
        hardware_model,
      },
    ];
  }
}
