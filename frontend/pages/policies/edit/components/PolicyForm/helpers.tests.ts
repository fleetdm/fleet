import { getPolicyAutomationErrorMessage } from "./helpers";

const createMockApiError = (reason: string) => ({
  data: { errors: [{ name: "base", reason }] },
});

describe("getPolicyAutomationErrorMessage", () => {
  it("passes through the API reason when a declaration was selected", () => {
    const reason = "Couldn't add. Fleet can't resend declaration (DDM).";
    expect(getPolicyAutomationErrorMessage(createMockApiError(reason))).toBe(
      reason
    );
  });

  it("rewrites the invalid profile prefix error", () => {
    expect(
      getPolicyAutomationErrorMessage(
        createMockApiError("profile_uuid has an invalid prefix")
      )
    ).toBe(
      "Only Apple and Windows configuration profiles are supported. Please select a valid profile."
    );
  });

  it("falls back to a generic message for other errors", () => {
    expect(
      getPolicyAutomationErrorMessage(createMockApiError("something else"))
    ).toBe("Could not update policy automations.");
  });
});
