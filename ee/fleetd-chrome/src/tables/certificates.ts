import Table from "./Table";

export default class TableChromeExtensions extends Table {
  name = "certificates";
  columns = ["token"];

  async generate() {
    let allCertificates = [];

    try {
      const tokens: any[] = await new Promise((resolve) =>
        chrome.enterprise.platformKeys.getTokens(resolve)
      );

      for (let i = 0; i < tokens.length; i++) {
        const certificates = await new Promise((resolve) =>
          chrome.enterprise.platformKeys.getCertificates(tokens[i], resolve)
        );
        allCertificates.push({ token: tokens[i], certificates });
      }
    } catch (err) {
      console.warn("get certificates info:", err);
    }

    return [{ allCertificates }];
  }
}
