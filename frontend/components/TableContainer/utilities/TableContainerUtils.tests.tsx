import { generateResultsCountText } from "./TableContainerUtils";

describe("generateResultsCountText", () => {
  it("handles 'teams', 'items' 'results' 'users' 'hosts' 'labels' correctly", () => {
    expect(generateResultsCountText("teams", 0)).toBe("0 teams");
    expect(generateResultsCountText("teams", 1)).toBe("1 team");
    expect(generateResultsCountText("teams", 2)).toBe("2 teams");

    expect(generateResultsCountText("items", 1)).toBe("1 item");
    expect(generateResultsCountText("items", 5)).toBe("5 items");

    expect(generateResultsCountText("results", 1)).toBe("1 result");
    expect(generateResultsCountText("results", 10)).toBe("10 results");

    expect(generateResultsCountText("users", 1)).toBe("1 user");
    expect(generateResultsCountText("users", 3)).toBe("3 users");

    expect(generateResultsCountText("hosts", 1)).toBe("1 host");
    expect(generateResultsCountText("hosts", 9)).toBe("9 hosts");

    expect(generateResultsCountText("labels", 1)).toBe("1 label");
    expect(generateResultsCountText("labels", 4)).toBe("4 labels");

    expect(generateResultsCountText("versions", 1)).toBe("1 version");
    expect(generateResultsCountText("versions", 7)).toBe("7 versions");
  });

  it("handles 'policies' and 'queries' correctly", () => {
    expect(generateResultsCountText("policies", 1)).toBe("1 policy");
    expect(generateResultsCountText("policies", 6)).toBe("6 policies");

    expect(generateResultsCountText("queries", 1)).toBe("1 query");
    expect(generateResultsCountText("queries", 2)).toBe("2 queries");
  });

  it("handles 'certificates' correctly", () => {
    expect(generateResultsCountText("certificates", 1)).toBe("1 certificate");
    expect(generateResultsCountText("certificates", 2)).toBe("2 certificates");
  });

  it("handles 'classes' correctly with singular form 'class' provided", () => {
    expect(generateResultsCountText("classes", 1, "class")).toBe("1 class");
    expect(generateResultsCountText("classes", 2, "class")).toBe("2 classes");
  });
});
