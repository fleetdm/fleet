import { postBridgeMessage } from "./fleetDesktopBridge";

describe("fleetDesktopBridge", () => {
  afterEach(() => {
    delete window.webkit;
  });

  it("no-ops when window.webkit is missing (browser dev context)", () => {
    expect(() => postBridgeMessage("ready")).not.toThrow();
  });

  it("posts a versioned message with the given action and payload", () => {
    const postMessage = jest.fn();
    window.webkit = {
      messageHandlers: { fleetDesktop: { postMessage } },
    };

    postBridgeMessage("resize", { height: 400 });

    expect(postMessage).toHaveBeenCalledWith({
      v: 1,
      action: "resize",
      payload: { height: 400 },
    });
  });

  it("defaults payload to an empty object", () => {
    const postMessage = jest.fn();
    window.webkit = {
      messageHandlers: { fleetDesktop: { postMessage } },
    };

    postBridgeMessage("ready");

    expect(postMessage).toHaveBeenCalledWith({
      v: 1,
      action: "ready",
      payload: {},
    });
  });
});
