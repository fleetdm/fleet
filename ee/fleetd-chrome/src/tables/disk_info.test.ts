import VirtualDatabase from "../db";

const INTERNAL_DISK_INFO_MOCK = {  
    estimate: {
      quota: 54321,
    }  
};
const INTERNAL_DISK_INFO_EXPECTED_RESULT = {  
    capacity: "54321",
    id: "internal",
    name: "Internal Storage",
    type: "fixed",
};

const REMOVABLE_DISK_INFO_MOCK = [
  {
    capacity: "1234",
    id: "123",
    name: "Cell phone (internal storage)",
    type: "removable",
  },
  {
    capacity: "0",
    id: "12",
    name: "Thumbdrive",
    type: "removable",
  },
];

describe("disk_info", () => {
  let orgNavigator = window.navigator;
  beforeAll(() => {
    // Mock the Navigator API.
    Object.defineProperty(window, 'navigator', {
      value: {storage: {
        estimate: () => Promise.resolve({ quota: 54321 })        
      }},
      writable: true
    })
  });
  afterAll(() => {
    // Restore the original Navigator API.
    window.navigator = orgNavigator;
  });

  test("success", async () => {
    // @ts-ignore
    chrome.system.storage.getInfo = jest.fn(() =>
      Promise.resolve(REMOVABLE_DISK_INFO_MOCK)
    );

    const db = await VirtualDatabase.init();

    const res = await db.query("select * from disk_info");
    expect(res).toEqual({"data":[INTERNAL_DISK_INFO_EXPECTED_RESULT, ...REMOVABLE_DISK_INFO_MOCK], "warnings": null});
  });
});
