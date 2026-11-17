import { ICertificateAuthorityPartial } from "interfaces/certificates";

import {
  CA_NAME_IN_USE_ERROR,
  CA_NAME_NDES_ERROR,
  getCANameConflictError,
  getConflictingCANames,
} from "./helpers";

const certAuthorities: ICertificateAuthorityPartial[] = [
  { id: 1, name: "NDES", type: "ndes_scep_proxy" },
  { id: 2, name: "WIFI", type: "digicert" },
  { id: 3, name: "VPN", type: "custom_scep_proxy" },
];

describe("getCANameConflictError", () => {
  it.each([
    ["a free name", "NEW_NAME", undefined, undefined],
    [
      "the name of a CA of another type",
      "VPN",
      undefined,
      CA_NAME_IN_USE_ERROR,
    ],
    ["a taken name in another case", "wifi", undefined, CA_NAME_IN_USE_ERROR],
    [
      "a taken name with spaces around it",
      " WIFI ",
      undefined,
      CA_NAME_IN_USE_ERROR,
    ],
    ["the edited CA's own name", "wifi", 2, undefined],
    ["another CA's name when editing", "VPN", 2, CA_NAME_IN_USE_ERROR],
    ["the NDES name in another case", "ndes", undefined, CA_NAME_NDES_ERROR],
    ["the NDES name, even when editing", "Ndes", 1, CA_NAME_NDES_ERROR],
  ])("checks %s", (_, name, editingId, expected) => {
    expect(getCANameConflictError(name, certAuthorities, editingId)).toBe(
      expected
    );
  });
});

describe("getConflictingCANames", () => {
  it("returns nothing when the names are unique", () => {
    expect(getConflictingCANames(certAuthorities)).toEqual([]);
  });

  it("groups the CAs that share a name without regard to case", () => {
    const legacy: ICertificateAuthorityPartial[] = [
      ...certAuthorities,
      { id: 4, name: "wifi", type: "hydrant" },
      { id: 5, name: "Vpn", type: "smallstep" },
    ];
    expect(getConflictingCANames(legacy)).toEqual([
      [certAuthorities[1], legacy[3]],
      [certAuthorities[2], legacy[4]],
    ]);
  });
});
