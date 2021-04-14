import formatApiErrors from "utilities/format_api_errors";

describe("formatApiErrors", () => {
  const errorStub = {
    response: {
      errors: [
        {
          name: "email",
          reason: "is not the correct format",
        },
        {
          name: "kolide_server_url",
          reason: "must be present",
        },
      ],
    },
  };

  it("formats errors for the Form HOC", () => {
    expect(formatApiErrors(errorStub)).toEqual({
      email: "is not the correct format",
      kolide_server_url: "must be present",
    });
  });
});
