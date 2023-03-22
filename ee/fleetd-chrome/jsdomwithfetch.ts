// Adapted from https://github.com/jsdom/jsdom/issues/1724#issuecomment-1446858041

import JSDOMEnvironment from "jest-environment-jsdom";

export default class JSDOMWithFetch extends JSDOMEnvironment {
  constructor(...args: ConstructorParameters<typeof JSDOMEnvironment>) {
    super(...args);

    // Fixes for missing fetch (https://github.com/jsdom/jsdom/issues/1724)
    this.global.fetch = fetch;
    this.global.Headers = Headers;
    this.global.Request = Request;
    this.global.Response = Response;
  }
}
