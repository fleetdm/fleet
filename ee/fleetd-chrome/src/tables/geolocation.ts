import Table from "./Table";

export default class TableGeolocation extends Table {
  name = "geolocation";
  columns = ["ip", "city", "country", "region"];

  async generate() {
    const resp = await fetch("https://ipapi.co/json");
    const json = await resp.json();
    return {
      data: [
        {
          ip: json.ip,
          city: json.city,
          country: json.country_name,
          region: json.region,
        },
      ],
    };
  }
}
