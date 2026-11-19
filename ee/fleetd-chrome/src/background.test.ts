describe("startup timestamp", () => {
  afterEach(() => jest.restoreAllMocks());

  test("records each startup without overwriting the timestamp on worker load", async () => {
    chrome.runtime.onStartup = { addListener: jest.fn() } as unknown as typeof chrome.runtime.onStartup;
    chrome.storage = {
      local: { set: jest.fn().mockResolvedValue(undefined) },
    } as unknown as typeof chrome.storage;
    chrome.alarms = {
      create: jest.fn(),
      onAlarm: { addListener: jest.fn() },
    } as unknown as typeof chrome.alarms;
    chrome.management = {
      getSelf: jest.fn().mockResolvedValue({ installType: "admin" }),
    } as unknown as typeof chrome.management;
    Object.defineProperty(navigator, "userAgentData", {
      configurable: true,
      value: { platform: "Linux" },
    });
    jest.spyOn(console, "error").mockImplementation(() => {});
    jest.spyOn(globalThis, "setTimeout").mockImplementation(() => 0 as unknown as ReturnType<typeof setTimeout>);
    jest.spyOn(Date, "now").mockReturnValue(1800000000000);

    await import("./background");

    expect(chrome.storage.local.set).not.toHaveBeenCalled();
    expect(chrome.runtime.onStartup.addListener).toHaveBeenCalledTimes(1);
    const onStartup = (chrome.runtime.onStartup.addListener as jest.Mock).mock.calls[0][0];
    await onStartup();
    expect(chrome.storage.local.set).toHaveBeenLastCalledWith({ lastRebootTimestamp: 1800000000000 });

    jest.spyOn(Date, "now").mockReturnValue(1800000100000);
    await onStartup();
    expect(chrome.storage.local.set).toHaveBeenLastCalledWith({ lastRebootTimestamp: 1800000100000 });
  });
});