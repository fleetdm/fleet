import { installCSPNonce } from "./install_csp_nonce";

describe("installCSPNonce", () => {
  const nativeCreateElement = document.createElement.bind(document);

  const setNonceMeta = (value: string | null) => {
    document.head.innerHTML = "";
    if (value !== null) {
      const meta = nativeCreateElement("meta");
      meta.setAttribute("property", "csp-nonce");
      meta.setAttribute("content", value);
      document.head.appendChild(meta);
    }
  };

  afterEach(() => {
    document.createElement = nativeCreateElement;
    document.head.innerHTML = "";
  });

  it("stamps the nonce on created <style> elements", () => {
    setNonceMeta("abc123");
    installCSPNonce();

    expect(document.createElement("style").getAttribute("nonce")).toEqual(
      "abc123"
    );
  });

  it("does not stamp <script> or unrelated elements", () => {
    setNonceMeta("abc123");
    installCSPNonce();

    expect(document.createElement("script").hasAttribute("nonce")).toBe(false);
    expect(document.createElement("div").hasAttribute("nonce")).toBe(false);
  });

  it("does nothing when no csp-nonce meta tag is present", () => {
    setNonceMeta(null);
    installCSPNonce();

    expect(document.createElement("style").hasAttribute("nonce")).toBe(false);
  });

  it("does nothing when the nonce is the disabled sentinel", () => {
    setNonceMeta("disabled");
    installCSPNonce();

    expect(document.createElement("style").hasAttribute("nonce")).toBe(false);
  });
});
