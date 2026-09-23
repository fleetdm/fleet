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
const IOS_USER_AGENT =
  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1";
const MACOS_USER_AGENT =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36";
const ENROLL_URL = "https://enterprise.google.com/android/enroll?et=test-token";
const TOKEN_ENDPOINT = "/api/v1/fleet/android_enterprise/enrollment_token";

// jsdom implements no navigation, so it reports attempts through the virtual
// console. Collecting them is the only way to observe location.assign and
// location.replace, whose own properties are unforgeable and cannot be stubbed.
const NAVIGATION_ERROR = "Not implemented: navigation (except hash changes)";

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
let jsdomErrors: string[];

/**
 * Loads the page and waits for its DOMContentLoaded handler to finish. `path`
 * and `search` decide which of the two pages is being simulated, matching what
 * the server mounts at /enroll and /enroll/next-steps.
 */
const loadPage = async ({
  userAgent = ANDROID_USER_AGENT,
  nextSteps = false,
  search = "?enroll_secret=test-secret",
  urlPrefix = "",
  androidMDMEnabled = "true",
  enrollmentUrl = ENROLL_URL as string | null,
} = {}) => {
  const html = renderTemplate({
    AndroidMDMEnabled: androidMDMEnabled,
    MacMDMEnabled: "true",
    AppleManualEnrollmentBlocked: "false",
    NextSteps: String(nextSteps),
    CSPNonce: "test-nonce",
    ErrorMessage: "",
    URLPrefix: urlPrefix,
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
  const pagePath = nextSteps ? "/enroll/next-steps" : "/enroll";
  dom = new JSDOM(html, {
    url: `https://fleet.example.com${urlPrefix}${pagePath}${search}`,
    runScripts: "outside-only",
    virtualConsole,
  });
  doc = dom.window.document;
  Object.defineProperty(dom.window.navigator, "userAgent", {
    configurable: true,
    get: () => userAgent,
  });

  const fetchMock = jest.fn().mockResolvedValue({
    ok: true,
    json: async () => ({ android_enrollment_url: enrollmentUrl }),
  });
  dom.window.fetch = (fetchMock as unknown) as typeof fetch;

  const inlineScript = Array.from(doc.querySelectorAll("script")).find(
    (script) => !script.src
  );
  // A top-level `const` inside eval is scoped to that eval and doesn't become a
  // global, so the page's URL builder is handed out explicitly to be asserted
  // on. This appends to the evaluated copy only; the template is untouched.
  dom.window.eval(
    `${inlineScript?.textContent ?? ""}
     window.__buildEnrollPath = buildEnrollPath;`
  );

  // jsdom queues its own DOMContentLoaded, so the page's handler is left to run
  // on that. Dispatching one here as well would run the handler twice and
  // double up its listeners. Two turns: one for the event, one for the token
  // fetch the handler awaits.
  await new Promise((resolve) => setTimeout(resolve, 0));
  await new Promise((resolve) => setTimeout(resolve, 0));

  return { fetchMock };
};

const clickEnroll = () => {
  const enrollLink = doc.querySelector<HTMLAnchorElement>(".enroll-link");
  expect(enrollLink).not.toBeNull();
  // Added after the page's own listener, so it runs second: the page navigates
  // this tab, and this stops jsdom also trying to follow the href.
  enrollLink?.addEventListener("click", (event) => event.preventDefault(), {
    once: true,
  });
  enrollLink?.dispatchEvent(
    new dom.window.MouseEvent("click", { bubbles: true, cancelable: true })
  );
};

const mainContentText = () =>
  doc.querySelector("#main-content")?.textContent ?? "";

const heading = () => doc.querySelector("#main-content h1")?.textContent;

// The page builds both of its own URLs through this helper.
const buildEnrollPath = (page: string): string =>
  ((dom.window as unknown) as {
    __buildEnrollPath: (page: string) => string;
  }).__buildEnrollPath(page);

afterEach(() => {
  dom?.window.close();
});

describe("enroll-ota.html — /enroll, Android", () => {
  it("renders the enroll screen with the enrollment URL from the API", async () => {
    const { fetchMock } = await loadPage();

    expect(fetchMock).toHaveBeenCalledWith(
      `${TOKEN_ENDPOINT}?enroll_secret=test-secret`,
      expect.anything()
    );
    expect(heading()).toBe("How to enroll your Android device to Fleet");
    expect(doc.querySelector<HTMLAnchorElement>(".enroll-link")?.href).toBe(
      ENROLL_URL
    );
    // The close-the-tab guidance now lives on the next-steps page.
    expect(mainContentText()).not.toContain("close this");
  });

  it("sends this tab to the next-steps page when Enroll is selected", async () => {
    await loadPage();
    expect(jsdomErrors).toEqual([]);

    clickEnroll();

    expect(jsdomErrors).toEqual([NAVIGATION_ERROR]);
    expect(buildEnrollPath("/enroll/next-steps")).toBe(
      "/enroll/next-steps?enroll_secret=test-secret"
    );
  });

  it("carries the enroll secret through, escaped, and honours a URL prefix", async () => {
    await loadPage({ search: "?enroll_secret=a%2Bb%20c", urlPrefix: "/fleet" });

    expect(buildEnrollPath("/enroll/next-steps")).toBe(
      "/fleet/enroll/next-steps?enroll_secret=a%2Bb%20c"
    );
  });

  it("doesn't wire the navigation when the token request fails", async () => {
    await loadPage({ enrollmentUrl: null });

    expect(doc.querySelector(".error-title")?.textContent).toBe(
      "Couldn't get Android enrollment token."
    );
    clickEnroll();
    expect(jsdomErrors).toEqual([]);
  });
});

describe("enroll-ota.html — /enroll/next-steps", () => {
  it("renders the next steps for Android and fetches nothing", async () => {
    const { fetchMock } = await loadPage({ nextSteps: true });

    expect(heading()).toBe("Next steps...");
    const steps = Array.from(
      doc.querySelectorAll("#main-content li")
    ).map((li) => li.textContent?.replace(/\s+/g, " ").trim());
    expect(steps).toEqual([
      "1. Follow enrollment instructions.",
      '2. When you see the "Your device is ready to go!" page, you\'re done.',
      "3. You can close this page.",
    ]);
    // Informational page: no token, and nothing left to select.
    expect(fetchMock).not.toHaveBeenCalled();
    expect(doc.querySelector(".enroll-link")).toBeNull();
    expect(jsdomErrors).toEqual([]);
  });

  it("redirects iOS back to the enroll page", async () => {
    await loadPage({ nextSteps: true, userAgent: IOS_USER_AGENT });

    expect(jsdomErrors).toEqual([NAVIGATION_ERROR]);
    expect(heading()).toBeUndefined();
    expect(buildEnrollPath("/enroll")).toBe(
      "/enroll?enroll_secret=test-secret"
    );
  });

  it("redirects macOS back to the enroll page", async () => {
    await loadPage({ nextSteps: true, userAgent: MACOS_USER_AGENT });

    expect(jsdomErrors).toEqual([NAVIGATION_ERROR]);
    expect(heading()).toBeUndefined();
  });

  it("redirects back without a query when there is no enroll secret", async () => {
    await loadPage({
      nextSteps: true,
      userAgent: MACOS_USER_AGENT,
      search: "",
    });

    expect(jsdomErrors).toEqual([NAVIGATION_ERROR]);
    expect(buildEnrollPath("/enroll")).toBe("/enroll");
  });

  it("shows the next steps rather than an error when the secret is absent", async () => {
    await loadPage({ nextSteps: true, search: "" });

    expect(heading()).toBe("Next steps...");
    expect(doc.querySelector(".error-title")).toBeNull();
  });
});
