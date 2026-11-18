import {
  FLEET_NAME_CONFLICT_ERROR,
  FLEET_NAME_NO_LETTER_OR_NUMBER_ERROR,
  getFleetNameError,
} from "./helpers";

const response = (status: number, name: string, reason: string) => ({
  status,
  data: { message: "error", errors: [{ name, reason }] },
});

describe("getFleetNameError", () => {
  it("returns the conflict error for a 409, whatever the reason says", () => {
    [
      'A fleet named "Workstations" already exists. Choose a fleet name distinct from other fleets.',
      'A fleet named "Workstations" already exists. Fleet names must differ by more than letter case.',
      "something else entirely",
    ].forEach((reason) => {
      expect(
        getFleetNameError(response(409, "base", reason), "work-stations")
      ).toBe(FLEET_NAME_CONFLICT_ERROR);
    });
    expect(getFleetNameError({ status: 409 }, "work-stations")).toBe(
      FLEET_NAME_CONFLICT_ERROR
    );
  });

  it("returns the letter-or-number error for that 422", () => {
    expect(
      getFleetNameError(
        response(422, "name", "must include at least one letter or number"),
        "🚀"
      )
    ).toBe(FLEET_NAME_NO_LETTER_OR_NUMBER_ERROR);
  });

  it("returns the reserved-name error with the name the user typed", () => {
    expect(
      getFleetNameError(
        response(422, "name", '"unassigned" is a reserved fleet name'),
        "unassigned"
      )
    ).toBe('"unassigned" is a reserved fleet name. Please try another name.');
    expect(
      getFleetNameError(
        response(422, "name", '"All fleets" is a reserved fleet name'),
        "All fleets"
      )
    ).toBe('"All fleets" is a reserved fleet name. Please try another name.');
  });

  it("returns undefined for other errors", () => {
    expect(
      getFleetNameError(response(422, "name", "may not be empty"), "")
    ).toBeUndefined();
    expect(
      getFleetNameError(
        response(422, "name", "may not exceed 255 characters"),
        "a"
      )
    ).toBeUndefined();
    expect(
      getFleetNameError(
        response(422, "secrets", "must include at least one letter or number"),
        "a"
      )
    ).toBeUndefined();
    expect(
      getFleetNameError(
        response(500, "name", "must include at least one letter or number"),
        "a"
      )
    ).toBeUndefined();
    expect(
      getFleetNameError(response(403, "base", "forbidden"), "a")
    ).toBeUndefined();
    expect(getFleetNameError({ status: 422, data: {} }, "a")).toBeUndefined();
  });

  it("returns undefined for a 422 whose body isn't a Fleet error", () => {
    [
      "<html>Unprocessable</html>",
      { errors: "must include at least one letter or number" },
      { errors: [null, "name", { name: "name" }, { name: "name", reason: 1 }] },
      null,
    ].forEach((data) => {
      expect(getFleetNameError({ status: 422, data }, "a")).toBeUndefined();
    });
    expect(getFleetNameError("Network Error", "a")).toBeUndefined();
    expect(getFleetNameError(undefined, "a")).toBeUndefined();
  });
});
