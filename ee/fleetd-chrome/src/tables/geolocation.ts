import Table from "./Table";

export default class TableGeolocation extends Table {
  name = "geolocation";
  columns = ["ip", "city", "country", "region"];

  ensureString(val: unknown): string {
    val = val ?? ""; // coerce undefined/null to empty string
    if (typeof val !== "string") {
      return val.toString();
    }
    return val;
  }

  async generate() {
    const resp = await fetch("https://ipapi.co/json");
    const json = await resp.json();
    return {
      data: [
        {
          ip: this.ensureString(json.ip),
          city: this.ensureString(json.city),
          country: this.ensureString(json.country_name),
          region: this.ensureString(json.region),
        },
      ],
    };
  }
}
