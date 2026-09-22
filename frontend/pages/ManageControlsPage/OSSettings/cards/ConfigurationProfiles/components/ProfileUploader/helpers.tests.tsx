import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { IMdmProfile } from "interfaces/mdm";

import {
  DEFAULT_EDIT_ERROR_MESSAGE,
  DEFAULT_ERROR_MESSAGE,
  generateCustomTargetLabelKey,
  getAcceptedExtensions,
  getErrorMessage,
  detectProfileContentType,
  parseFile,
  editorModeForContentType,
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
    // an XML declaration is read as a plist, so this isn't Windows
    expect(detectProfileContentType('<?xml version="1.0"?><Replace/>')).toBe(
      "mobileconfig"
    );
    // SyncML has to start with a command or a comment
    expect(detectProfileContentType("<!-- firewall --><Add/>")).toBe("windows");
    expect(detectProfileContentType("<SyncML><Replace/></SyncML>")).toBeNull();
    // only a top-level Apple Type makes a declaration
    expect(
      detectProfileContentType(
        '{"applications": [{"Type": "com.apple.not.top.level"}]}'
      )
    ).toBe("android");
    expect(detectProfileContentType("{not json")).toBeNull();
    expect(detectProfileContentType("[]")).toBeNull();
  });

  it("accepts uppercase file extensions", async () => {
    const parsed = await parseFile(
      new File(["<plist/>"], "Policy.MOBILECONFIG", { type: "text/plain" })
    );
    expect(parsed.ext).toBe("mobileconfig");
    expect(parsed.name).toBe("Policy");
  });

  it("maps types to editor modes", () => {
    expect(editorModeForContentType("mobileconfig")).toBe("xml");
    expect(editorModeForContentType("windows")).toBe("xml");
    expect(editorModeForContentType("declaration")).toBe("json");
    expect(editorModeForContentType("android")).toBe("json");
    expect(editorModeForContentType(null)).toBe("text");
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
