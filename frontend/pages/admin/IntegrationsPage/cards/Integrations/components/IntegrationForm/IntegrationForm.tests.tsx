import { validateForm } from "./IntegrationForm";

describe("IntegrationForm validateForm", () => {
  const data = {
    url: "https://example.atlassian.net",
    apiToken: "token",
    email: "user@example.com",
    groupId: 1,
  };

  it("accepts a valid URL and email with surrounding whitespace", () => {
    const errors = validateForm(
      { ...data, url: " " + data.url + " ", email: " " + data.email + " " },
      "zendesk"
    );

    expect(errors.url).toBeUndefined();
    expect(errors.email).toBeUndefined();
  });

  it("still rejects an invalid URL", () => {
    const errors = validateForm({ ...data, url: " not a url " }, "zendesk");

    expect(errors.url).toBe("Enter a valid HTTPS URL");
  });
});
