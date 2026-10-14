import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { IMdmProfile } from "interfaces/mdm";

import {
  contentTypeForExtension,
  DEFAULT_EDIT_ERROR_MESSAGE,
  DEFAULT_ERROR_MESSAGE,
  generateCustomTargetLabelKey,
  getAcceptedExtensions,
  getErrorMessage,
  detectProfileContentType,
  parseFile,
  PROFILE_CONTENT_TYPES,
  profileContentTypeFor,
  nextPastedProfileName,
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

describe("detectProfileContentType", () => {
  it("types pasted text by its shape", () => {
    expect(
      detectProfileContentType(
        '<?xml version="1.0"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "x"><plist version="1.0"><dict/></plist>'
      )
    ).toBe("mobileconfig");
    expect(
      detectProfileContentType('  <plist version="1.0"><dict/></plist>')
    ).toBe("mobileconfig");
    expect(detectProfileContentType("<Replace><Item/></Replace>")).toBe(
      "windows"
    );
    expect(
      detectProfileContentType(
        '{"Type": "com.apple.configuration.passcode.settings", "Identifier": "x"}'
      )
    ).toBe("declaration");
    expect(detectProfileContentType('{"cameraDisabled": true}')).toBe(
      "android"
    );
    expect(detectProfileContentType("not a profile")).toBeNull();
    expect(detectProfileContentType("")).toBeNull();
  });

  it("follows the server's rules for ambiguous text", () => {
    // SyncML behind an XML declaration is Windows, which the server rejects
    // with an error that names the declaration
    expect(detectProfileContentType('<?xml version="1.0"?>\n<Replace/>')).toBe(
      "windows"
    );
    // a plist with a comment after its declaration stays a mobileconfig
    expect(
      detectProfileContentType(
        '<?xml version="1.0"?><!-- exported --><!DOCTYPE plist PUBLIC "x" "y"><plist version="1.0"/>'
      )
    ).toBe("mobileconfig");
    // any other XML declaration is read as a plist
    expect(detectProfileContentType('<?xml version="1.0"?><foo/>')).toBe(
      "mobileconfig"
    );
    // SyncML has to start with a command or a comment
    expect(detectProfileContentType("<!-- firewall --><Add/>")).toBe("windows");
    expect(detectProfileContentType("<Exec><Item/></Exec>")).toBe("windows");
    expect(detectProfileContentType("<SyncML><Replace/></SyncML>")).toBeNull();
    // JSON a secret placeholder keeps from parsing is a declaration
    expect(
      detectProfileContentType(
        '{"Type": "com.apple.configuration.passcode.settings", "Payload": $FLEET_SECRET_PASSCODE}'
      )
    ).toBe("declaration");
    // a lone placeholder says nothing about the type
    expect(detectProfileContentType("$FLEET_SECRET_PROFILE")).toBeNull();
    // top-level key casing decides, as on the server; nested keys don't count
    expect(
      detectProfileContentType(
        '{"applications": [{"Type": "com.apple.not.top.level"}]}'
      )
    ).toBe("android");
    // uppercase keys without a Type still read as a declaration, so the
    // server can report the missing Type
    expect(detectProfileContentType('{"Identifier": "x", "Payload": {}}')).toBe(
      "declaration"
    );
    // mixed casing is rejected by the server; a Type points at a declaration
    expect(
      detectProfileContentType('{"Type": "com.apple.x", "payload": {}}')
    ).toBe("declaration");
    expect(
      detectProfileContentType('{"Name": "x", "cameraDisabled": true}')
    ).toBe("android");
    // a key starting with a digit is neither case; the server rejects it
    // either way, so this only picks the gate its error comes back through
    expect(detectProfileContentType('{"1something": true}')).toBe("android");
    expect(
      detectProfileContentType('{"Type": "com.apple.x", "1something": true}')
    ).toBe("declaration");
    expect(detectProfileContentType("{not json")).toBeNull();
    expect(detectProfileContentType("[]")).toBeNull();
  });

  it("types an uploaded file by its extension as the server does", () => {
    expect(contentTypeForExtension("mobileconfig")).toBe("mobileconfig");
    expect(contentTypeForExtension("xml")).toBe("windows");
    expect(contentTypeForExtension("json")).toBe("declaration");
    expect(contentTypeForExtension("txt")).toBeNull();
  });

  it("accepts uppercase file extensions", async () => {
    const parsed = await parseFile(
      new File(["<plist/>"], "Policy.MOBILECONFIG", { type: "text/plain" })
    );
    expect(parsed.ext).toBe("mobileconfig");
    expect(parsed.name).toBe("Policy");
  });

  it("edits XML types as xml and JSON types as json", () => {
    expect(PROFILE_CONTENT_TYPES.mobileconfig.editorMode).toBe("xml");
    expect(PROFILE_CONTENT_TYPES.windows.editorMode).toBe("xml");
    expect(PROFILE_CONTENT_TYPES.declaration.editorMode).toBe("json");
    expect(PROFILE_CONTENT_TYPES.android.editorMode).toBe("json");
  });

  it("derives an existing profile's type from the API shape", () => {
    const base = {
      team_id: 0,
      name: "n",
      identifier: null,
      created_at: "",
      updated_at: "",
      checksum: null,
    };
    const profile = (overrides: Partial<IMdmProfile>): IMdmProfile =>
      ({ ...base, ...overrides } as IMdmProfile);
    expect(
      profileContentTypeFor(
        profile({ profile_uuid: "d-1", platform: "darwin" })
      )
    ).toBe("declaration");
    expect(
      profileContentTypeFor(
        profile({ profile_uuid: "a-1", platform: "darwin" })
      )
    ).toBe("mobileconfig");
    expect(
      profileContentTypeFor(
        profile({ profile_uuid: "w-1", platform: "windows" })
      )
    ).toBe("windows");
    expect(
      profileContentTypeFor(
        profile({ profile_uuid: "g-1", platform: "android" })
      )
    ).toBe("android");
  });
});

describe("getErrorMessage duplicate name", () => {
  it("prefixes the duplicate name error with the flow", () => {
    const reason =
      "A configuration profile with this name already exists. Enter a different name.";
    expect(
      getErrorMessage({
        data: { errors: [{ name: "profile", reason }] },
      } as AxiosResponse<IApiError>)
    ).toEqual(`Couldn't add. ${reason}`);
    expect(
      getErrorMessage(
        {
          data: { errors: [{ name: "profile", reason }] },
        } as AxiosResponse<IApiError>,
        "edit"
      )
    ).toEqual(`Couldn't edit. ${reason}`);
  });

  it("keeps the server's own flow prefix", () => {
    const reason =
      "Couldn't edit. A configuration profile with this name already exists. Enter a different name.";
    expect(
      getErrorMessage(
        {
          data: { errors: [{ name: "profile", reason }] },
        } as AxiosResponse<IApiError>,
        "edit"
      )
    ).toEqual(reason);
  });
});

describe("nextPastedProfileName", () => {
  it("starts at New profile, then numbers from 2", () => {
    expect(nextPastedProfileName([])).toBe("New profile");
    expect(nextPastedProfileName(["Wi-Fi"])).toBe("New profile");
    expect(nextPastedProfileName(["New profile"])).toBe("New profile 2");
    expect(nextPastedProfileName(["New profile", "New profile 2"])).toBe(
      "New profile 3"
    );
  });

  it("fills the lowest gap", () => {
    expect(nextPastedProfileName(["New profile", "New profile 3"])).toBe(
      "New profile 2"
    );
  });

  it("compares without case or surrounding spaces", () => {
    expect(nextPastedProfileName([" new profile ", "NEW PROFILE 2"])).toBe(
      "New profile 3"
    );
  });
});

describe("getAcceptedExtensions", () => {
  const profile = (
    profileUUID: string,
    platform: IMdmProfile["platform"]
  ): IMdmProfile => ({
    profile_uuid: profileUUID,
    team_id: 0,
    name: "n",
    platform,
    identifier: null,
    created_at: "",
    updated_at: "",
    checksum: null,
    self_service: false,
    hidden: false,
  });

  it.each([
    [
      "a mobileconfig, which may be saved as .xml",
      profile("a-1", "darwin"),
      [".mobileconfig", ".xml"],
    ],
    ["a declaration", profile("d-1", "darwin"), [".json"]],
    ["a Windows profile", profile("w-1", "windows"), [".xml"]],
    ["an Android profile", profile("g-1", "android"), [".json"]],
  ])("accepts the right files for %s", (_, p, expected) => {
    expect(getAcceptedExtensions(p)).toEqual(expected);
  });
});
