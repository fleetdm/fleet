import fs from "fs";
import path from "path";

import { JSDOM, VirtualConsole } from "jsdom";

/**
 * enroll-ota.html is a Go template served straight to the device, so there is
 * no bundler entry point to import. These tests substitute the template
 * variables the server fills in, load the page into its own JSDOM window, and
 * run its inline script by hand. Each test gets a fresh window so the script's
 * top-level declarations and listeners don't leak between them.
 */

const ANDROID_USER_AGENT =
  "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36";
const ENROLL_URL = "https://enterprise.google.com/android/enroll?et=test-token";
const TOKEN_ENDPOINT = "/api/v1/fleet/android_enterprise/enrollment_token";

const templateSource = fs.readFileSync(
  path.join(__dirname, "enroll-ota.html"),
  "utf8"
);

const renderTemplate = (values: Record<string, string>) =>
  templateSource.replace(
    /{{\.(\w+)}}/g,
    (_match, name: string) => values[name] ?? ""
  );

let dom: JSDOM;
let doc: Document;
// jsdom has no navigation, so it reports attempts through the virtual console.
// Collecting them is the only way to observe location.reload(), whose own
// properties are unforgeable and therefore impossible to stub.
let jsdomErrors: string[];

const setVisibility = (state: "visible" | "hidden") => {
  Object.defineProperty(doc, "visibilityState", {
    configurable: true,
    get: () => state,
  });
  doc.dispatchEvent(new dom.window.Event("visibilitychange"));
};

/**
 * Loads the page for an Android device with a valid enroll secret and waits for
 * its DOMContentLoaded handler to finish rendering.
 */
const loadAndroidPage = async ({
  androidMDMEnabled = "true",
  enrollmentUrl = ENROLL_URL as string | null,
} = {}) => {
  const html = renderTemplate({
    AndroidMDMEnabled: androidMDMEnabled,
    MacMDMEnabled: "false",
    AppleManualEnrollmentBlocked: "false",
    CSPNonce: "test-nonce",
    ErrorMessage: "",
    URLPrefix: "",
    EnrollURL: "https://fleet.example.com/enroll",
  });

  jsdomErrors = [];
  const virtualConsole = new VirtualConsole();
  virtualConsole.on("jsdomError", (error: Error) => {
    jsdomErrors.push(error.message);
  });

  // outside-only leaves the page's own <script> tags inert (including the
  // qrcode CDN tag), so the stubs below are in place before the page script
  // runs via window.eval.
  dom = new JSDOM(html, {
    url: "https://fleet.example.com/enroll/ota?enroll_secret=test-secret",
    runScripts: "outside-only",
    virtualConsole,
  });
  doc = dom.window.document;
  Object.defineProperty(dom.window.navigator, "userAgent", {
    configurable: true,
    get: () => ANDROID_USER_AGENT,
  });
  setVisibility("visible");

  const fetchMock = jest.fn().mockResolvedValue({
    ok: true,
    json: async () => ({ android_enrollment_url: enrollmentUrl }),
  });
  dom.window.fetch = (fetchMock as unknown) as typeof fetch;

  const inlineScript = Array.from(doc.querySelectorAll("script")).find(
    (script) => !script.src
  );
  dom.window.eval(inlineScript?.textContent ?? "");

  doc.dispatchEvent(new dom.window.Event("DOMContentLoaded"));
  // Let the handler's awaited token fetch settle.
  await new Promise((resolve) => setTimeout(resolve, 0));

  return { fetchMock };
};

const clickEnroll = () => {
  const enrollLink = doc.querySelector<HTMLAnchorElement>(".enroll-link");
  expect(enrollLink).not.toBeNull();
  // jsdom can't follow the enrollment link, and the page doesn't need it to.
  enrollLink?.addEventListener("click", (event) => event.preventDefault(), {
    once: true,
  });
  enrollLink?.dispatchEvent(
    new dom.window.MouseEvent("click", { bubbles: true, cancelable: true })
  );
};

const mainContentText = () =>
  doc.querySelector("#main-content")?.textContent ?? "";

const isPostEnrollScreen = () =>
  doc.querySelector("#main-content h1")?.textContent === "Already finished?";

describe("enroll-ota.html — Android", () => {
  it("renders the enroll screen with the enrollment URL from the API", async () => {
    const { fetchMock } = await loadAndroidPage();

    expect(fetchMock).toHaveBeenCalledWith(
      `${TOKEN_ENDPOINT}?enroll_secret=test-secret`,
      expect.anything()
    );
    expect(doc.querySelector("#main-content h1")?.textContent).toBe(
      "How to enroll your Android device to Fleet"
    );
    expect(doc.querySelector<HTMLAnchorElement>(".enroll-link")?.href).toBe(
      ENROLL_URL
    );
    // The close-the-tab guidance now lives on its own screen.
    expect(mainContentText()).not.toContain("close this tab");
  });

  it("swaps to the post-enroll screen once enrollment backgrounds the tab", async () => {
    await loadAndroidPage();

    clickEnroll();
    setVisibility("hidden");

    expect(isPostEnrollScreen()).toBe(true);
    expect(mainContentText()).toContain("You can close this tab.");
    expect(doc.querySelector(".enroll-link")).toBeNull();
  });

  it("keeps the enroll screen while the tab is still in the foreground", async () => {
    await loadAndroidPage();

    clickEnroll();
    // A visibilitychange that doesn't hide the page must not swap the content
    // out from under the user while they're reading it.
    setVisibility("visible");

    expect(isPostEnrollScreen()).toBe(false);
    expect(doc.querySelector(".enroll-link")).not.toBeNull();
  });

  it("keeps the enroll screen when the tab is hidden without selecting Enroll", async () => {
    await loadAndroidPage();

    setVisibility("hidden");

    expect(isPostEnrollScreen()).toBe(false);
    expect(doc.querySelector(".enroll-link")).not.toBeNull();
  });

  it("only swaps once, so returning to a hidden tab doesn't re-render", async () => {
    await loadAndroidPage();

    clickEnroll();
    setVisibility("hidden");
    setVisibility("visible");
    setVisibility("hidden");

    expect(doc.querySelectorAll("#main-content h1")).toHaveLength(1);
    expect(isPostEnrollScreen()).toBe(true);
  });

  it("reloads the page from the post-enroll refresh link", async () => {
    await loadAndroidPage();

    clickEnroll();
    setVisibility("hidden");
    expect(jsdomErrors).toEqual([]);

    const refreshLink = doc.querySelector<HTMLAnchorElement>(".refresh-link");
    expect(refreshLink).not.toBeNull();
    const event = new dom.window.MouseEvent("click", {
      bubbles: true,
      cancelable: true,
    });
    refreshLink?.dispatchEvent(event);

    // The reload, and the href="#" left unfollowed.
    expect(jsdomErrors).toEqual([expect.stringContaining("navigation")]);
    expect(event.defaultPrevented).toBe(true);
  });

  it("doesn't wire the post-enroll screen when the token request fails", async () => {
    await loadAndroidPage({ enrollmentUrl: null });

    setVisibility("hidden");

    expect(isPostEnrollScreen()).toBe(false);
    expect(doc.querySelector(".error-title")?.textContent).toBe(
      "Couldn't get Android enrollment token."
    );
  });
});
