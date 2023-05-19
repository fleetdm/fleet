import { enforceFleetSentenceCasing } from "./stringUtils";

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
