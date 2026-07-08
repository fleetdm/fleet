import {
  dateAgo,
  monthDayYearFormat,
  addedFromNow,
  uploadedFromNow,
  monthDayTimeFormat,
  timeAgo,
} from ".";

describe("date_format utilities", () => {
  describe("uploadedFromNow util", () => {
    it("returns an user friendly uploaded message", () => {
      const currentDate = new Date();
      currentDate.setDate(currentDate.getDate() - 2);
      const twoDaysAgo = currentDate.toISOString();

      expect(uploadedFromNow(twoDaysAgo)).toEqual("Uploaded 2 days ago");
    });
  });

  describe("addedFromNow util", () => {
    it("returns an user friendly added message", () => {
      const currentDate = new Date();
      currentDate.setDate(currentDate.getDate() - 2);
      const twoDaysAgo = currentDate.toISOString();

      expect(addedFromNow(twoDaysAgo)).toEqual("Added 2 days ago");
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

    const daysAgo = (n: number) =>
      new Date(Date.now() - n * 24 * 60 * 60 * 1000).toISOString();

    it("uses days below the month threshold", () => {
      expect(dateAgo(daysAgo(5))).toEqual("5 days ago");
      expect(dateAgo(daysAgo(29))).toEqual("29 days ago");
      expect(dateAgo(daysAgo(30))).toEqual("30 days ago");
      expect(dateAgo(daysAgo(40))).toEqual("40 days ago");
      expect(dateAgo(daysAgo(60))).toEqual("60 days ago");
      expect(dateAgo(daysAgo(89))).toEqual("89 days ago");
    });

    it("uses months at or beyond 90 days", () => {
      expect(dateAgo(daysAgo(90))).toEqual("3 months ago");
      expect(dateAgo(daysAgo(100))).toEqual("3 months ago");
    });
  });

  describe("timeAgo util", () => {
    const daysAgo = (n: number) =>
      new Date(Date.now() - n * 24 * 60 * 60 * 1000);

    it("omits the `ago` suffix by default and adds it when requested", () => {
      expect(timeAgo(daysAgo(40))).toEqual("40 days");
      expect(timeAgo(daysAgo(40), { addSuffix: true })).toEqual("40 days ago");
      expect(timeAgo(daysAgo(89), { addSuffix: true })).toEqual("89 days ago");
    });

    it("switches to months at 90 days", () => {
      expect(timeAgo(daysAgo(90), { addSuffix: true })).toEqual("3 months ago");
    });

    // strict avoids the "about" prefix outside the day window.
    it("supports the strict variant outside the window", () => {
      expect(timeAgo(daysAgo(100), { addSuffix: true, strict: true })).toEqual(
        "3 months ago"
      );
    });
  });

  describe("monthDayTimeFormat util", () => {
    it("returns a formatted date string matching pattern 'Mon D, H:MM AM/PM'", () => {
      const date = "2024-03-20T13:35:00Z";
      const result = monthDayTimeFormat(date);
      // Match pattern like "Mar 20, 1:35 PM" (exact time varies by timezone)
      expect(result).toMatch(/^[A-Z][a-z]{2} \d{1,2}, \d{1,2}:\d{2} (AM|PM)$/);
    });

    it("returns an empty string for invalid dates", () => {
      expect(monthDayTimeFormat("invalid-date")).toEqual("");
    });
  });
});
