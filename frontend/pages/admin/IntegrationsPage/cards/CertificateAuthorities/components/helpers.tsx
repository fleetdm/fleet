import {
  ICertificateAuthorityPartial,
  ICertificateAuthorityType,
} from "interfaces/certificates";

const CA_LABEL_BY_TYPE: Record<ICertificateAuthorityType, string> = {
  custom_est_proxy: "Custom Enrollment Over Secure Transport (EST)",
  custom_scep_proxy: "Custom Simple Certificate Enrollment Protocol (SCEP)",
  digicert: "DigiCert",
  ejbca: "EJBCA",
  google_cas: "Google Certificate Authority Service (CAS)",
  hydrant: "Hydrant Enrollment Over Secure Transport (EST)",
  ndes_scep_proxy:
    "Dynamic SCEP - Okta CA or Microsoft Network Device Enrollment Service (NDES)",
  smallstep: "Smallstep",
};

export default CA_LABEL_BY_TYPE;

export const CA_NAME_NDES_ERROR = "Enter a name other than NDES";
export const CA_NAME_IN_USE_ERROR =
  "Enter a name that isn't used by another certificate authority";

/**
 * Returns the error for a CA name that is reserved or taken. Names are unique across all CA types,
 * without regard to case, and the NDES CA's name is fixed.
 */
export const getCANameConflictError = (
  name: string,
  certAuthorities: ICertificateAuthorityPartial[],
  editingId?: number
): string | undefined => {
  const lowerName = name.trim().toLowerCase();
  if (lowerName === "ndes") {
    return CA_NAME_NDES_ERROR;
  }
  if (
    certAuthorities.some(
      (ca) => ca.id !== editingId && ca.name.toLowerCase() === lowerName
    )
  ) {
    return CA_NAME_IN_USE_ERROR;
  }
  return undefined;
};

/**
 * Returns the groups of CAs that share a name, without regard to case. They can only exist from
 * before names had to be unique, and none of them can be edited until all but one are renamed.
 */
export const getConflictingCANames = (
  certAuthorities: ICertificateAuthorityPartial[]
): ICertificateAuthorityPartial[][] => {
  const byName = new Map<string, ICertificateAuthorityPartial[]>();
  certAuthorities.forEach((ca) => {
    const key = ca.name.toLowerCase();
    byName.set(key, [...(byName.get(key) ?? []), ca]);
  });
  return [...byName.values()].filter((group) => group.length > 1);
};
