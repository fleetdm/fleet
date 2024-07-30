import TableSystemInfo from "./system_info";

describe("system_info", () => {
  describe("getComputerName", () => {
    const sut = new TableSystemInfo(null, null);
    it("is computed from the hostname and hw serial", () => {
      const testCases: [string, string, string][] = [
        [null, null, "Chromebook"],
        [undefined, undefined, "Chromebook"],
        ["", "", "Chromebook"],
        ["mychromebook", "", "mychromebook"],
        ["mychromebook", "123", "mychromebook"],
        ["", "123", "Chromebook 123"],
      ];

      for (let [hostname, hwSerial, expected] of testCases) {
        expect(sut.getComputerName(hostname, hwSerial)).toEqual(expected);
      }
    });
  });
});
