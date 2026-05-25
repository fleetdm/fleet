import getDeleteLabelErrorMessages from "./helpers";

const make422 = (reason: string) => ({
  status: 422,
  errors: [{ name: "label", reason }],
});

describe("getDeleteLabelErrorMessages", () => {
  it("returns built-in message for built-in label errors", () => {
    expect(
      getDeleteLabelErrorMessages(make422("cannot delete built-in label"))
    ).toBe("Built-in labels can't be modified or deleted.");
  });

  it("returns configuration profile message when label is used by a profile", () => {
    expect(
      getDeleteLabelErrorMessages(make422("used by a configuration profile"))
    ).toBe(
      "Couldn't delete. A configuration profile targets this label. Please delete the profile and try again."
    );
  });

  it("returns software message for other 422 errors", () => {
    expect(
      getDeleteLabelErrorMessages(make422("used by software target"))
    ).toBe(
      "Couldn't delete. Software uses this label as a custom target. Remove the label from the software target and try again."
    );
  });

  it("returns generic message for non-422 errors", () => {
    expect(getDeleteLabelErrorMessages(new Error("network error"))).toBe(
      "Could not delete label. Please try again."
    );
  });

  it("returns generic message for unknown error shapes", () => {
    expect(getDeleteLabelErrorMessages({ status: 500 })).toBe(
      "Could not delete label. Please try again."
    );
  });
});
