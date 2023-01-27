import Table from "./Table.js";

export default class TableGeolocation extends Table {
  name = "geolocation";
  columns = ["ip", "city", "country", "region"];

  async generate(...args) {
    console.log("args", args);
    const resp = await fetch("https://ipapi.co/json");
    const json = await resp.json();
    console.log(json);
    return [[json.ip, json.city, json.country_name, json.region]];
  }
}
