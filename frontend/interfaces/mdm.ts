// TODO: Correct interface once backend is done

export interface IAPN {
  commonName: string;
  serialNumber: string;
  issuer: string;
  renewDate: string;
}

export interface IABM {
  team?: string;
  appleId: string;
  organizationName: string;
  mdmServerUrl: string;
  renewDate: string;
}

export interface IAppleMdm {
  apn: IAPN;
  abm: IABM;
}
