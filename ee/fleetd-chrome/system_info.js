import Table from "./Table.js";

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

  async generate(...args) {
    const uuid = await chrome.instanceID.getID();

    // TODO should it default to UUID or should Fleet handle it somehow?
    let hostname = uuid;
    try {
      hostname = await chrome.enterprise.deviceAttributes.getDeviceHostname();
    } catch (err) {
      console.error("get hostname:", err);
    }

    let hardware_serial = "";
    try {
      hardware_serial = await chrome.enterprise.deviceAttributes.getDeviceSerialNumber();
    } catch (err) {
      console.error("get serial number:", err);
    }

    let hardware_vendor = "",
      hardware_model = "";
    try {
      // TODO figure out why this throws "Not allowed" error on test device
      const platform_info = await chrome.enterprise.hardwarePlatform.getHardwarePlatformInfo();
      hardware_vendor = platform_info.manufacturer;
      hardware_model = platform_info.model;
    } catch (err) {
      console.error("get platform info:", err);
    }

    return [
      [
        uuid,
        hostname,
        hostname,
        hardware_serial,
        hardware_vendor,
        hardware_model,
      ],
    ];
  }
}
