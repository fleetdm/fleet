import getCSPNonce from "./nonce";

describe("getCSPNonce", () => {
  afterEach(() => {
    document.head.innerHTML = "";
  });

  it("returns an empty string when the csp-nonce meta tag is absent", () => {
    expect(getCSPNonce()).toEqual("");
  });

  it("returns the nonce from the csp-nonce meta tag when present", () => {
    const meta = document.createElement("meta");
    meta.setAttribute("property", "csp-nonce");
    meta.setAttribute("content", "abc123");
    document.head.appendChild(meta);

    expect(getCSPNonce()).toEqual("abc123");
  });

  it("returns an empty string when the meta tag has no content", () => {
    const meta = document.createElement("meta");
    meta.setAttribute("property", "csp-nonce");
    document.head.appendChild(meta);

    expect(getCSPNonce()).toEqual("");
  });
});
