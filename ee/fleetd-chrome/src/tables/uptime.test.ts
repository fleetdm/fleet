import VirtualDatabase from "../db";

describe("uptime", () => {
  const now = 1800000000000;

  beforeEach(() => {
    jest.spyOn(Date, "now").mockReturnValue(now);
    chrome.storage = {
      local: { get: jest.fn(), set: jest.fn() },
    } as unknown as typeof chrome.storage;
  });

  afterEach(() => jest.restoreAllMocks());

  test("returns elapsed time in osquery-compatible columns", async () => {
    (chrome.storage.local.get as jest.Mock).mockResolvedValue({
      lastRebootTimestamp: now - (2 * 86400 + 3 * 3600 + 4 * 60 + 5) * 1000 - 999,
    });
    const db = await VirtualDatabase.init();

    expect(await db.query("SELECT * FROM uptime")).toEqual({
      data: [{ days: "2", hours: "3", minutes: "4", seconds: "5", total_seconds: "183845" }],
      warnings: null,
    });
    expect(chrome.storage.local.get).toHaveBeenCalledWith("lastRebootTimestamp");
    expect(chrome.storage.local.set).not.toHaveBeenCalled();
  });

  test.each([undefined, null, "invalid", 0, NaN, Infinity])(
    "returns no rows for an unavailable timestamp (%s)",
    async (lastRebootTimestamp) => {
      (chrome.storage.local.get as jest.Mock).mockResolvedValue({ lastRebootTimestamp });
      const db = await VirtualDatabase.init();

      expect(await db.query("SELECT * FROM uptime")).toEqual({ data: [], warnings: null });
    }
  );

  test("does not return negative uptime when the clock moves backwards", async () => {
    (chrome.storage.local.get as jest.Mock).mockResolvedValue({ lastRebootTimestamp: now + 1000 });
    const db = await VirtualDatabase.init();

    expect(await db.query("SELECT * FROM uptime")).toEqual({
      data: [{ days: "0", hours: "0", minutes: "0", seconds: "0", total_seconds: "0" }],
      warnings: null,
    });
  });

  test("uses the persisted timestamp across database instances", async () => {
    (chrome.storage.local.get as jest.Mock).mockResolvedValue({ lastRebootTimestamp: now - 1000 });
    const firstDB = await VirtualDatabase.init();
    expect((await firstDB.query("SELECT total_seconds FROM uptime")).data).toEqual([{ total_seconds: "1" }]);

    jest.spyOn(Date, "now").mockReturnValue(now + 1000);
    const secondDB = await VirtualDatabase.init();
    expect((await secondDB.query("SELECT total_seconds FROM uptime")).data).toEqual([{ total_seconds: "2" }]);
    expect(chrome.storage.local.set).not.toHaveBeenCalled();
  });
});