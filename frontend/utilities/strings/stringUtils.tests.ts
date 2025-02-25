import {
  enforceFleetSentenceCasing,
  pluralize,
  strToBool,
  stripQuotes,
  isIncompleteQuoteQuery,
} from "./stringUtils";

describe("string utilities", () => {
  describe("enforceFleetSentenceCasing utility", () => {
    it("fixes a Title Cased String with no ignore words", () => {
      expect(enforceFleetSentenceCasing("All Hosts")).toEqual("All hosts");
      expect(enforceFleetSentenceCasing("all Hosts")).toEqual("All hosts");
      expect(enforceFleetSentenceCasing("all hosts")).toEqual("All hosts");
      expect(enforceFleetSentenceCasing("All HosTs ")).toEqual("All hosts");
    });

    it("fixes a title cased string while ignoring special words in various places ", () => {
      expect(enforceFleetSentenceCasing("macOS")).toEqual("macOS");
      expect(enforceFleetSentenceCasing("macOS Settings")).toEqual(
        "macOS settings"
      );
      expect(
        enforceFleetSentenceCasing("osquery shouldn't be Capitalized")
      ).toEqual("osquery shouldn't be capitalized");
    });
    expect(enforceFleetSentenceCasing("fleet uses MySQL")).toEqual(
      "Fleet uses MySQL"
    );
  });

  describe("pluralize utility", () => {
    it("returns the singular form of a word when count is 1", () => {
      expect(pluralize(1, "hero", "es", "")).toEqual("hero");
    });

    it("returns the plural form of a word when count is not 1", () => {
      expect(pluralize(0, "hero", "es", "")).toEqual("heroes");
      expect(pluralize(2, "hero", "es", "")).toEqual("heroes");
      expect(pluralize(100, "hero", "es", "")).toEqual("heroes");
    });

    it("returns the singular form of a word when count is 1 and a no custom suffix are provided", () => {
      expect(pluralize(1, "hero")).toEqual("hero");
    });

    it("returns the pluralized form of a word with 's' suffix when count is not 1 and no custom suffix are provided", () => {
      expect(pluralize(0, "hero")).toEqual("heros");
      expect(pluralize(2, "hero")).toEqual("heros");
      expect(pluralize(100, "hero")).toEqual("heros");
    });
  });

  describe("strToBool utility", () => {
    it("converts 'true' to true and 'false' to false", () => {
      expect(strToBool("true")).toBe(true);
      expect(strToBool("false")).toBe(false);
    });

    it("returns false for undefined, null, or empty string", () => {
      expect(strToBool(undefined)).toBe(false);
      expect(strToBool(null)).toBe(false);
      expect(strToBool("")).toBe(false);
    });
  });

  describe("stripQuotes utility", () => {
    it("removes matching single or double quotes from the start and end of a string", () => {
      expect(stripQuotes('"Hello, World!"')).toEqual("Hello, World!");
      expect(stripQuotes("'Hello, World!'")).toEqual("Hello, World!");
    });
    it("does not modify a string without quotes or mismatched quotes", () => {
      expect(stripQuotes("No quotes here")).toEqual("No quotes here");
      expect(stripQuotes(`'Mismatched quotes"`)).toEqual(`'Mismatched quotes"`);
    });
  });

  describe("isIncompleteQuoteQuery utility", () => {
    it("returns true for a string starting with a quote but not ending with one", () => {
      expect(isIncompleteQuoteQuery('"incomplete')).toBe(true);
      expect(isIncompleteQuoteQuery("'incomplete")).toBe(true);
    });

    it("returns false for a string with matching quotes", () => {
      expect(isIncompleteQuoteQuery('"complete"')).toBe(false);
      expect(isIncompleteQuoteQuery("'complete'")).toBe(false);
    });

    it("returns false for a string without any quotes or an empty string", () => {
      expect(isIncompleteQuoteQuery("no quotes")).toBe(false);
      expect(isIncompleteQuoteQuery("")).toBe(false);
    });
  });
});
