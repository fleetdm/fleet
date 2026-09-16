import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";

import {
  DEFAULT_EDIT_ERROR_MESSAGE,
  DEFAULT_ERROR_MESSAGE,
  generateCustomTargetLabelKey,
  getErrorMessage,
} from "./helpers";

describe("generateCustomTargetLabelKey", () => {
  it("returns empty object when target is not Custom", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "All hosts",
        includeMode: "any",
        includeLabels: { foo: true },
        excludeLabels: {},
      })
    ).toEqual({});
  });

  it("returns labelsIncludeAny when include mode is any", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "any",
        includeLabels: { foo: true, bar: true },
        excludeLabels: {},
      })
    ).toEqual({ labelsIncludeAny: ["foo", "bar"] });
  });

  it("returns labelsIncludeAll when include mode is all", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "all",
        includeLabels: { foo: true },
        excludeLabels: {},
      })
    ).toEqual({ labelsIncludeAll: ["foo"] });
  });

  it("returns labelsExcludeAny when exclude labels are selected", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "any",
        includeLabels: {},
        excludeLabels: { bar: true },
      })
    ).toEqual({ labelsExcludeAny: ["bar"] });
  });

  it("returns both include and exclude keys when both have selections", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "all",
        includeLabels: { foo: true },
        excludeLabels: { bar: true },
      })
    ).toEqual({ labelsIncludeAll: ["foo"], labelsExcludeAny: ["bar"] });
  });

  it("omits keys for empty selections", () => {
    expect(
      generateCustomTargetLabelKey({
        targetType: "Custom",
        includeMode: "all",
        includeLabels: { foo: false },
        excludeLabels: {},
      })
    ).toEqual({});
  });
});

const createErrResponse = (reason: string) =>
  (({
    data: { message: "Bad request", errors: [{ name: "base", reason }] },
  } as unknown) as AxiosResponse<IApiError>);

describe("getErrorMessage", () => {
  it("returns the add default message when there is no api reason", () => {
    expect(getErrorMessage(createErrResponse(""))).toEqual(
      DEFAULT_ERROR_MESSAGE
    );
  });

  it("returns the edit default message when there is no api reason and action is edit", () => {
    expect(getErrorMessage(createErrResponse(""), "edit")).toEqual(
      DEFAULT_EDIT_ERROR_MESSAGE
    );
  });

  it("returns the api reason verbatim when it isn't specially handled", () => {
    const reason =
      "profiles managed by Fleet can't be edited using this endpoint.";
    expect(getErrorMessage(createErrResponse(reason), "edit")).toEqual(reason);
  });

  it("maps the .mobileconfig PayloadIdentifier mismatch error", () => {
    expect(
      getErrorMessage(
        createErrResponse(
          "The new profile's PayloadIdentifier must match the existing profile's."
        ),
        "edit"
      )
    ).toEqual(
      "Couldn't edit. The uploaded profile must have the same PayloadIdentifier as the original profile."
    );
  });

  it("maps the declaration (DDM) identifier mismatch error", () => {
    expect(
      getErrorMessage(
        createErrResponse(
          "The new profile's Identifier must match the existing profile's."
        ),
        "edit"
      )
    ).toEqual(
      "Couldn't edit. The uploaded profile must have the same identifier as the original profile."
    );
  });

  it("maps the Windows/Android name mismatch error", () => {
    expect(
      getErrorMessage(
        createErrResponse(
          "The new profile's name must match the existing profile's name."
        ),
        "edit"
      )
    ).toEqual(
      "Couldn't edit. The uploaded profile must have the same name as the original profile."
    );
  });

  it('prefixes known validation messages with "Couldn\'t add." for the add flow', () => {
    expect(
      getErrorMessage(
        createErrResponse("The profile should include valid JSON")
      )
    ).toEqual("Couldn't add. The profile should include valid JSON.");
  });

  it('prefixes known validation messages with "Couldn\'t edit." for the edit flow', () => {
    expect(
      getErrorMessage(
        createErrResponse("The profile should include valid JSON"),
        "edit"
      )
    ).toEqual("Couldn't edit. The profile should include valid JSON.");
  });

  it("rephrases the OS updates error for the edit flow", () => {
    const reason =
      "Couldn't add profile. OS updates are already configured. Remove the OS updates settings first.";
    expect(getErrorMessage(createErrResponse(reason), "edit")).toEqual(
      "Couldn't edit profile. OS updates are already configured. Remove the OS updates settings first."
    );
    expect(getErrorMessage(createErrResponse(reason))).toEqual(reason);
  });
});
