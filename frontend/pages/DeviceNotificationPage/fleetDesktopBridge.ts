import { BridgeAction, IBridgeMessage } from "interfaces/device_notification";

// Fleet Desktop's WKWebView injects `window.webkit.messageHandlers.fleetDesktop`.
// The same page opened in a normal browser during development does not have it,
// so every bridge call must guard for the handler being absent.
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
    return;
  }
  const message: IBridgeMessage = { v: 1, action, payload };
  handler.postMessage(message);
};
