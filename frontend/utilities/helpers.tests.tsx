import { addCommasToCount } from "./helpers";

describe("addCommasToCount", () => {
  it("Correctly adds commas to numbers", () => {
    const ins = [1, 10, 100, 1000, 10000, 100000, 1000000];
    const outs = ["1", "10", "100", "1,000", "10,000", "100,000", "1,000,000"];
    ins.forEach((inVal, i) => {
      expect(addCommasToCount(inVal)).toEqual(outs[i]);
    });
  });
});
