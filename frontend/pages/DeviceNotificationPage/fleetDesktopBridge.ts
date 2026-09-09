import {
  BRIDGE_VERSION,
  BridgeAction,
  IBridgeMessage,
} from "interfaces/device_notification";

// The WKWebView handler is absent in a normal browser (dev), so guard every call.
declare global {
  interface Window {
    webkit?: {
      messageHandlers?: {
        fleetDesktop?: {
          postMessage: (message: unknown) => void;
        };
      };
    };
  }
}

// eslint-disable-next-line import/prefer-default-export
export const postBridgeMessage = (
  action: BridgeAction,
  payload: Record<string, unknown> = {}
): void => {
  const handler = window.webkit?.messageHandlers?.fleetDesktop;
  if (!handler) {
    if (process.env.NODE_ENV !== "production") {
      // eslint-disable-next-line no-console
      console.debug("[fleet-desktop-bridge] window.webkit handler missing", {
        action,
        payload,
      });
    }
    return;
  }
  const message: IBridgeMessage = { v: BRIDGE_VERSION, action, payload };
  handler.postMessage(message);
};
