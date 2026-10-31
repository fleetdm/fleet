import { validate } from "./OktaConditionalAccessModal";

describe("OktaConditionalAccessModal validate", () => {
  const data = {
    okta_idp_id: "idp",
    okta_acs_url: "https://example.okta.com/acs",
    okta_audience_uri: "audience",
    okta_certificate: "cert",
  };

  it("accepts a valid ACS URL with surrounding whitespace", () => {
    const errors = validate({
      ...data,
      okta_acs_url: " " + data.okta_acs_url + " ",
    });

    expect(errors.okta_acs_url).toBeUndefined();
  });

  it("still rejects an invalid ACS URL", () => {
    const errors = validate({ ...data, okta_acs_url: " not a url " });

    expect(errors.okta_acs_url).toBe(
      "Enter a valid URL with http or https scheme"
    );
  });
});
