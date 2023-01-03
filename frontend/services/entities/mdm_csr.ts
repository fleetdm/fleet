import { SetStateAction } from "react";
/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
// import { sendRequest } from "services/mock_service/service/service"; // MDM TODO: Replace when backend is merged
import { IRequestCSRFormData } from "interfaces/request_csr";
import axios, { AxiosError } from "axios";
import local from "utilities/local";

// This API call is made to a specific endpoint that is different than our
// other ones. This is why we have implmented the call with axios here instead
// of using our sendRequest method.
const requestCSR = async (
  formData: IRequestCSRFormData,
  setRequestState: React.Dispatch<
    SetStateAction<"loading" | "error" | "success" | undefined>
  >
) => {
  setRequestState("loading");
  const token = local.getItem("auth_token");
  const url = "https://www.fleetdm.com/api/v1/get_signed_apns_csr";
  try {
    const { data } = await axios({
      method: "post",
      url,
      data: formData,
      responseType: "json",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  } catch (error) {
    const axiosError = error as AxiosError;
    return axiosError.response;
  }
};

// const requestCSR = (formData: IRequestCSRFormData) => {
//   alert("made it!");
//   const destination = "https://www.fleetdm.com//api/v1/get_signed_apns_csr";
//   return sendRequest(
//     (method = "POST"),
//     (customDestination = destination),
//     (data = formData)
//   );
// };

export default requestCSR;
