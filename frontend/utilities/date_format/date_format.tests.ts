import { uploadedFromNow } from ".";

describe("date_format", () => {
  describe("uploadedFromNow util", () => {
    it("returns an user friendly uploaded at message", () => {
      const currentDate = new Date();
      currentDate.setDate(currentDate.getDate() - 2);
      const twoDaysAgo = currentDate.toISOString();

      expect(uploadedFromNow(twoDaysAgo)).toEqual("Uploaded 2 days ago");
    });
  });
});
