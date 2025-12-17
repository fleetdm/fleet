import { buildQueryStringFromParams } from "utilities/url";
import endpoints from "utilities/endpoints";
import sendRequest from "services";
import {
  ICertificateAuthorityPartial,
  ICertificateAuthority,
  ICertificatesCustomSCEP,
  ICertificatesDigicert,
  ICertificatesHydrant,
  ICertificatesNDES,
  ICertificatesSmallstep,
  ICertificatesCustomEST,
} from "interfaces/certificates";
import {
  ListEntitiesResponsePaginationCommon,
  PaginationParams,
} from "./common";

type IGetCertAuthoritiesListResponse = {
  certificate_authorities: ICertificateAuthorityPartial[];
};

type IAddCertAuthorityResponse = ICertificateAuthorityPartial;

type IGetCertAuthorityResponse = ICertificateAuthority;

interface IRequestCertAuthorityResponse {
  certificate: string;
}

export type IAddCertAuthorityBody =
  | { digicert: ICertificatesDigicert }
  | { ndes_scep_proxy: ICertificatesNDES }
  | { custom_scep_proxy: ICertificatesCustomSCEP }
  | { hydrant: ICertificatesHydrant }
  | { smallstep: ICertificatesSmallstep }
  | { custom_est_proxy: ICertificatesCustomEST };

export type IEditCertAuthorityBody =
  | { digicert: Partial<ICertificatesDigicert> }
  | { ndes_scep_proxy: Partial<ICertificatesNDES> }
  | { custom_scep_proxy: Partial<ICertificatesCustomSCEP> }
  | { hydrant: Partial<ICertificatesHydrant> }
  | { smallstep: Partial<ICertificatesSmallstep> }
  | { custom_est_proxy: Partial<ICertificatesCustomEST> };

interface IGetCertsParams extends PaginationParams {
  // not supported: after, order key, order direction, match query, meta (always included)
  team_id?: number;
}

export interface IQueryKeyGetCerts extends IGetCertsParams {
  scope: "certificates";
}
export interface ICertificate {
  id: number;
  name: string;
  certificate_authority_id: number;
  certificate_authority_name: string;
  created_at: string;
}
export interface IGetCertsResponse {
  meta: ListEntitiesResponsePaginationCommon;
  certificates: ICertificate[];
}

export interface ICreateCert {
  name: string;
  certAuthorityId: number;
  subjectName: string;
  teamId?: number;
}

export default {
  getCertificateAuthoritiesList: (): Promise<IGetCertAuthoritiesListResponse> => {
    const { CERTIFICATE_AUTHORITIES } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITIES);
  },

  getCertificateAuthority: (id: number): Promise<IGetCertAuthorityResponse> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITY(id));
  },

  addCertificateAuthority: (
    certData: IAddCertAuthorityBody
  ): Promise<IAddCertAuthorityResponse> => {
    const { CERTIFICATE_AUTHORITIES } = endpoints;
    return sendRequest("POST", CERTIFICATE_AUTHORITIES, certData);
  },

  editCertificateAuthority: (
    id: number,
    updateData: IEditCertAuthorityBody
  ): Promise<void> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("PATCH", CERTIFICATE_AUTHORITY(id), updateData);
  },

  deleteCertificateAuthority: (id: number): Promise<void> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("DELETE", CERTIFICATE_AUTHORITY(id));
  },

  requestCertificate: (id: number): Promise<IRequestCertAuthorityResponse> => {
    const { CERTIFICATE_AUTHORITY_REQUEST_CERT } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITY_REQUEST_CERT(id));
  },
  getCerts: ({
    team_id,
    page,
    per_page,
  }: IGetCertsParams): Promise<IGetCertsResponse> => {
    const { CERTIFICATES } = endpoints;

    const queryString = buildQueryStringFromParams({ team_id, page, per_page });

    return sendRequest(
      "GET",
      queryString ? CERTIFICATES.concat(`?${queryString}`) : CERTIFICATES
    );
  },
  createCert: ({ name, certAuthorityId, subjectName, teamId }: ICreateCert) => {
    const { CERTIFICATES } = endpoints;
    const requestBody = {
      name,
      certificate_authority_id: certAuthorityId,
      subject_name: subjectName,
    };
    return sendRequest(
      "POST",
      teamId ? CERTIFICATES.concat(`?team_id=${teamId}`) : CERTIFICATES,
      requestBody
    );
  },
  deleteCert: (id: number) => {
    return sendRequest("DELETE", endpoints.CERTIFICATES.concat(`/${id}`));
  },
};
