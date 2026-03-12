// Encapsulate the URL Prefix so that this is the only module that
// needs to access the global. All other modules should use this one.

// extending window interface to include urlPrefix.
// https://stackoverflow.com/questions/12709074/how-do-you-explicitly-set-a-new-property-on-window-in-typescript
declare global {
  interface Window {
    urlPrefix: string;
  }
}

export default window.urlPrefix || "";
