import { enforceFleetSentenceCasing, pluralize } from "./stringUtils";

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
});
