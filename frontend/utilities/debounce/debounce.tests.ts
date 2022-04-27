import debounce from "./index";

describe("debounce - utility", () => {
  it("prevents double-clicks from executing a function multiple times", () => {
    let count = 0;
    const increaseCount = () => {
      count += 1;
    };
    const debouncedFunc = debounce(increaseCount);

    debouncedFunc();
    debouncedFunc();
    expect(count).toEqual(1);
  });
});
