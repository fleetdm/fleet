import { dateAgo, monthDayYearFormat, uploadedFromNow } from ".";

describe("date_format utilities", () => {
  describe("uploadedFromNow util", () => {
    it("returns an user friendly uploaded at message", () => {
      const currentDate = new Date();
      currentDate.setDate(currentDate.getDate() - 2);
      const twoDaysAgo = currentDate.toISOString();

      expect(uploadedFromNow(twoDaysAgo)).toEqual("Uploaded 2 days ago");
    });
  });

  describe("monthDayYearFormat util", () => {
    it("returns a date in the format of 'MonthName Date, Year' (e.g. January 01, 2024)", () => {
      const date = "2024-11-29T00:00:00Z";
      expect(monthDayYearFormat(date)).toEqual("November 29, 2024");
    });
  });

  describe("dateAgo util", () => {
    it("returns a user friendly date ago message from a date string", () => {
      const currentDate = new Date();
      currentDate.setDate(currentDate.getDate() - 2);
      const twoDaysAgo = currentDate.toISOString();

      expect(dateAgo(twoDaysAgo)).toEqual("2 days ago");
    });

    it("returns a user friendly date ago message from a Date object", () => {
      const date = new Date();
      date.setDate(date.getDate() - 2);

      expect(dateAgo(date)).toEqual("2 days ago");
    });
  });
});
