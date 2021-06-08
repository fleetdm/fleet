import { convertSeconds } from "./helpers";

describe("Timer convertSeconds helper", () => {
  it("converts seconds to hh:mm:ss", () => {
    expect(convertSeconds(3000)).toEqual("00:00:03");
    expect(convertSeconds(30000)).toEqual("00:00:30");
    expect(convertSeconds(300000)).toEqual("00:05:00");
    expect(convertSeconds(0)).toEqual("00:00:00");
  });
});
