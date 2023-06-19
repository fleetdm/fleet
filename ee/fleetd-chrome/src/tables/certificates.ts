import Table from "./Table";

export default class TableCertificates extends Table {
  name = "certificates";
  columns = ["token"];

  async generate() {
    let allCertificates;

    const tokens: any[] = await new Promise((resolve) =>
      chrome.enterprise.platformKeys.getTokens(resolve)
    );

    for (let i = 0; i < tokens.length; i++) {
      const certificates = await new Promise((resolve) =>
        chrome.enterprise.platformKeys.getCertificates(tokens[i].id, resolve)
      );
      allCertificates.push({ token: tokens[i], certificates });
    }

    return [{ allCertificates }];
  }
}
