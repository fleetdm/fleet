import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";

import {
  DEFAULT_EDIT_ERROR_MESSAGE,
  DEFAULT_ERROR_MESSAGE,
  EXAMPLE_CUSTOM_ACTIVATION,
  generateCustomTargetLabelKey,
  getErrorMessage,
  parseFile,
} from "./helpers";

const jsonFile = (contents: string, name = "profile.json") =>
  new File([contents], name, { type: "application/json" });

const DDM_DECLARATION = JSON.stringify({
  Type: "com.apple.configuration.management.test",
  Identifier: "com.fleetdm.config.test",
  Payload: { Echo: "test" },
});

describe("parseFile", () => {
  it("identifies an Apple DDM declaration and extracts its identifier", async () => {
    const result = await parseFile(jsonFile(DDM_DECLARATION));

    expect(result).toMatchObject({
      name: "profile",
      ext: "json",
      isAppleDeclaration: true,
      declarationIdentifier: "com.fleetdm.config.test",
    });
  });

  it("does not treat an Android profile as a declaration", async () => {
    // Android profiles use lower-case keys and upload as .json, the same
    // extension as a declaration -- only the contents tell them apart.
    const result = await parseFile(
      jsonFile(JSON.stringify({ screenCaptureDisabled: true }))
    );

    expect(result.isAppleDeclaration).toBe(false);
    expect(result.declarationIdentifier).toBeUndefined();
  });

  it("reports a declaration without an Identifier, leaving the identifier unset", async () => {
    // the backend rejects this on upload; the UI still has to render an
    // example activation for it.
    const result = await parseFile(
      jsonFile(
        JSON.stringify({ Type: "com.apple.configuration.x", Payload: {} })
      )
    );

    expect(result.isAppleDeclaration).toBe(true);
    expect(result.declarationIdentifier).toBeUndefined();
  });

  it("requires an Apple Type, not just any Type key", async () => {
    const result = await parseFile(
      jsonFile(JSON.stringify({ Type: "com.example.thing", Payload: {} }))
    );

    expect(result.isAppleDeclaration).toBe(false);
  });

  it("rejects a non-string Type", async () => {
    const result = await parseFile(
      jsonFile(JSON.stringify({ Type: 42, Payload: {} }))
    );

    expect(result.isAppleDeclaration).toBe(false);
  });

  it("requires Payload to be an object", async () => {
    // the backend treats a scalar Payload as malformed.
    const result = await parseFile(
      jsonFile(
        JSON.stringify({ Type: "com.apple.configuration.x", Payload: "test" })
      )
    );

    expect(result.isAppleDeclaration).toBe(false);
  });

  it("rejects mixed casing, which the backend won't accept as either type", async () => {
    const result = await parseFile(
      jsonFile(
        JSON.stringify({
          Type: "com.apple.configuration.x",
          Payload: {},
          screenCaptureDisabled: true,
        })
      )
    );

    expect(result.isAppleDeclaration).toBe(false);
  });

  it("ignores a non-string Identifier", async () => {
    const result = await parseFile(
      jsonFile(
        JSON.stringify({
          Type: "com.apple.configuration.x",
          Identifier: 42,
          Payload: {},
        })
      )
    );

    expect(result.declarationIdentifier).toBeUndefined();
  });

  it("requires both Type and Payload to call it a declaration", async () => {
    const noPayload = await parseFile(
      jsonFile(JSON.stringify({ Type: "com.apple.configuration.x" }))
    );
    const noType = await parseFile(jsonFile(JSON.stringify({ Payload: {} })));

    expect(noPayload.isAppleDeclaration).toBe(false);
    expect(noType.isAppleDeclaration).toBe(false);
  });

  it("treats unparseable JSON as not a declaration rather than throwing", async () => {
    // the backend reports the parse error on upload -- the UI just needs to
    // keep the DDM-only section hidden.
    await expect(parseFile(jsonFile("{ not json"))).resolves.toMatchObject({
      isAppleDeclaration: false,
    });
  });

  it("treats a JSON array as not a declaration", async () => {
    await expect(
      parseFile(jsonFile(JSON.stringify([{ Type: "x", Payload: {} }])))
    ).resolves.toMatchObject({ isAppleDeclaration: false });
  });

  it("never flags non-JSON profile types as declarations", async () => {
    const mobileconfig = await parseFile(
      new File(["<plist></plist>"], "profile.mobileconfig")
    );
    const windows = await parseFile(new File(["<Replace></Replace>"], "p.xml"));

    expect(mobileconfig).toMatchObject({
      ext: "mobileconfig",
      isAppleDeclaration: false,
    });
    expect(windows).toMatchObject({ ext: "xml", isAppleDeclaration: false });
  });

  it("rejects an unsupported extension", async () => {
    await expect(parseFile(new File(["x"], "profile.txt"))).rejects.toThrow(
      "Invalid file type: txt"
    );
  });
});

describe("EXAMPLE_CUSTOM_ACTIVATION", () => {
  it("matches the activation shape shown in the design", () => {
    expect(JSON.parse(EXAMPLE_CUSTOM_ACTIVATION)).toEqual({
      Type: "com.apple.activation.simple",
      Identifier: "myIdentifier",
      Payload: {
        StandardConfigurations: ["myConfigurationIdentifier"],
      },
    });
  });

  it("renders indented JSON so the editor is readable", () => {
    expect(EXAMPLE_CUSTOM_ACTIVATION).toContain('\n  "Type"');
  });
});

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
