import { rest } from "msw";
import VirtualDatabase from "../db";
import { server } from "../mocks/server.js";

describe("geolocation", () => {
  let db: VirtualDatabase;
  beforeAll(async () => {
    db = await VirtualDatabase.init();
  });

  test("success", async () => {
    server.use(
      rest.get("https://ipapi.co/json", (req, res, ctx) => {
        return res(
          ctx.json({
            ip: "260f:1337:4a7e:e300:abcd:a98a:1234:18c",
            network: "260f:1337:4a78::/45",
            version: "IPv6",
            city: "Vancouver",
            region: "British Columbia",
            region_code: "BC",
            country: "CA",
            country_name: "Canada",
            country_code: "CA",
            country_code_iso3: "CAN",
            country_capital: "Ottawa",
            country_tld: ".ca",
            continent_code: "NA",
            in_eu: false,
            postal: "V5K",
            latitude: 49.282,
            longitude: -123.04,
            timezone: "America/Vancouver",
            utc_offset: "-0700",
            country_calling_code: "+1",
            currency: "CAD",
            currency_name: "Dollar",
            languages: "en-CA,fr-CA,iu",
            country_area: 9984670.0,
            country_population: 37058856,
            asn: "AS6327",
            org: "SHAW",
          })
        );
      })
    );
    const rows = await db.query("select * from geolocation");
    expect(rows).toEqual({
      data: [
        {
          ip: "260f:1337:4a7e:e300:abcd:a98a:1234:18c",
          city: "Vancouver",
          country: "Canada",
          region: "British Columbia",
        },
      ],
      warnings: null,
    });
  });

  test("request returns incomplete data", async () => {
    server.use(
      rest.get("https://ipapi.co/json", (req, res, ctx) => {
        return res(
          ctx.json({
            city: "Vancouver",
          })
        );
      })
    );
    const rows = await db.query("select * from geolocation");
    expect(rows).toEqual({
      data: [
        {
          ip: null,
          city: "Vancouver",
          country: null,
          region: null,
        },
      ],
      warnings: null,
    });
  });

  test("request fails", async () => {
    server.use(
      rest.get("https://ipapi.co/json", (req, res, ctx) => {
        return res(ctx.status(500));
      })
    );
    await expect(async () => {
      await db.query("select * from geolocation");
    }).rejects.toThrow();
  });
});
