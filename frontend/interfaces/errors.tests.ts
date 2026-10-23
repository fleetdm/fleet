import { splitServerErrors } from "./errors";

describe("splitServerErrors", () => {
  const nameMap = {
    tokenUrl: "mdm.apple_account_provisioning.oauth_idp_token_url",
    clientId: "mdm.apple_account_provisioning.oauth_idp_client_id",
  };
  const errors = [
    {
      name: "mdm.apple_account_provisioning.oauth_idp_token_url",
      reason: "must be a valid https URL",
    },
    { name: "mdm.apple_account_provisioning", reason: "Missing private key" },
  ];

  it.each([
    ["a sendRequest rejection", { status: 422, data: { errors } }],
    ["an axios-style error", { response: { data: { errors } } }],
    ["a bare error list", { errors }],
  ])("splits field errors from other reasons in %s", (_, payload) => {
    expect(splitServerErrors(payload, nameMap)).toEqual({
      fieldErrors: { tokenUrl: "must be a valid https URL" },
      otherReasons: ["Missing private key"],
    });
  });

  it("keeps the first error for a field and skips empty reasons", () => {
    const payload = {
      data: {
        errors: [
          { name: nameMap.clientId, reason: "" },
          { name: nameMap.tokenUrl, reason: "first" },
          { name: nameMap.tokenUrl, reason: "second" },
          { name: "base", reason: "" },
        ],
      },
    };
    expect(splitServerErrors(payload, nameMap)).toEqual({
      fieldErrors: { tokenUrl: "first" },
      otherReasons: [],
    });
  });

  it.each([
    ["no errors", { status: 500 }],
    ["an unparseable payload", "network down"],
    ["undefined", undefined],
  ])("returns nothing for %s", (_, payload) => {
    expect(splitServerErrors(payload, nameMap)).toEqual({
      fieldErrors: {},
      otherReasons: [],
    });
  });
});
