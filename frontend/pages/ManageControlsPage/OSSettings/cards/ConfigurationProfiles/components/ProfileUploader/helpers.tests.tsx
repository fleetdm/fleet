import { AxiosResponse } from "axios";
import { render, screen } from "@testing-library/react";

import { IApiError } from "interfaces/errors";

import {
  DEFAULT_EDIT_ERROR_MESSAGE,
  DEFAULT_ERROR_MESSAGE,
  decodeCustomActivation,
  EXAMPLE_CUSTOM_ACTIVATION,
  generateCustomTargetLabelKey,
  getErrorMessage,
  parseFile,
  validateCustomActivation,
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

  it("rejects an array Payload, which typeof reports as an object", async () => {
    const result = await parseFile(
      jsonFile(
        JSON.stringify({ Type: "com.apple.configuration.x", Payload: [] })
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
      Identifier: "01234567-ABCD-EFGH-IJKL-0123456789AB",
      Payload: {
        StandardConfigurations: ["01234567-ABCD-EFGH-IJKL-0123456789YZ"],
      },
    });
  });

  it("renders indented JSON so the editor is readable", () => {
    expect(EXAMPLE_CUSTOM_ACTIVATION).toContain('\n  "Type"');
  });
});

describe("decodeCustomActivation", () => {
  const encode = (value: string) => {
    const bytes = new TextEncoder().encode(value);
    let binary = "";
    bytes.forEach((byte) => {
      binary += String.fromCharCode(byte);
    });
    return btoa(binary);
  };

  it("returns an empty string when there's nothing to decode", () => {
    // a declaration using Fleet's generated activation has no activation field.
    expect(decodeCustomActivation(undefined)).toEqual("");
    expect(decodeCustomActivation("")).toEqual("");
  });

  it("decodes base64 raw JSON", () => {
    const activation = JSON.stringify({
      Type: "com.apple.activation.simple",
      Identifier: "com.fleetdm.activation.test",
    });

    expect(decodeCustomActivation(encode(activation))).toEqual(activation);
  });

  it("preserves the formatting the activation was uploaded with", () => {
    // the editor should show the admin their own layout, not a reformatted one.
    const indented = `{\n  "Type": "com.apple.activation.simple"\n}`;

    expect(decodeCustomActivation(encode(indented))).toEqual(indented);
  });

  it("decodes non-ASCII characters without mangling them", () => {
    // atob alone yields a binary string, which would corrupt these.
    const activation = '{"Identifier":"café-日本-\\u00e9"}';

    expect(decodeCustomActivation(encode(activation))).toEqual(activation);
  });
});

describe("validateCustomActivation", () => {
  const VALID_ACTIVATION = JSON.stringify({
    Type: "com.apple.activation.simple",
    Identifier: "com.fleetdm.activation.test",
    Payload: { StandardConfigurations: ["com.fleetdm.config.test"] },
  });

  it("accepts an empty value, since Fleet synthesizes an activation", () => {
    expect(validateCustomActivation("")).toBeNull();
    expect(validateCustomActivation("   \n ")).toBeNull();
  });

  it("accepts a complete activation", () => {
    expect(validateCustomActivation(VALID_ACTIVATION)).toBeNull();
  });

  it("accepts the example shown in the editor", () => {
    expect(validateCustomActivation(EXAMPLE_CUSTOM_ACTIVATION)).toBeNull();
  });

  it("rejects unparseable JSON", () => {
    expect(validateCustomActivation('{"Type"')).toBe("Enter valid JSON");
  });

  it("rejects JSON that isn't an object", () => {
    // JSON.parse accepts these, but neither can carry the required keys.
    expect(validateCustomActivation("123")).toBe("Enter a JSON object");
    expect(validateCustomActivation("[]")).toBe("Enter a JSON object");
    expect(validateCustomActivation("null")).toBe("Enter a JSON object");
  });

  it("reports a missing Type before checking its value", () => {
    expect(validateCustomActivation('{"Identifier": "x"}')).toBe(
      "Missing Type key"
    );
  });

  it("rejects a Type outside the activation namespace", () => {
    const configurationType = JSON.stringify({
      Type: "com.apple.configuration.management.test",
      Identifier: "com.fleetdm.activation.test",
    });

    expect(validateCustomActivation(configurationType)).toBe(
      "Type is invalid (must be com.apple.activation.*)"
    );
  });

  it("rejects a non-string Type without throwing", () => {
    expect(validateCustomActivation('{"Type": 123, "Identifier": "x"}')).toBe(
      "Type is invalid (must be com.apple.activation.*)"
    );
  });

  it("reports a missing Identifier once the Type is valid", () => {
    expect(
      validateCustomActivation('{"Type": "com.apple.activation.simple"}')
    ).toBe("Missing Identifier key");
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

  it("passes the referenced-configuration error through verbatim", () => {
    // the API phrases custom activation errors as complete sentences, so they
    // need no prefixing or rewording.
    const reason =
      "Couldn't add custom activation. The referenced configuration com.fleetdm.config.passcode doesn't exist. Please add the referenced configuration profile and try again.";
    expect(getErrorMessage(createErrResponse(reason))).toEqual(reason);
  });

  it("renders the trailing link on the one-configuration error", () => {
    const reason =
      "Couldn't add custom activation. The custom activation can only have one referenced configuration profile. Learn more: https://fleetdm.com/learn-more-about/ddm-activations";

    render(getErrorMessage(createErrResponse(reason)) as JSX.Element);

    expect(
      screen.getByText(
        "Couldn't add custom activation. The custom activation can only have one referenced configuration profile."
      )
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Learn more/ })).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/ddm-activations"
    );
  });
});
