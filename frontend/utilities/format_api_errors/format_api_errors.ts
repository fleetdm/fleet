import { IError } from "interfaces/errors";

export default (error: any) => {
  if (!error.response || !error.response.errors) {
    return undefined;
  }

  const { errors: errorsArray } = error.response;
  const result: { [key: string]: any } = {};

  errorsArray.forEach((errorObject: IError) => {
    result[errorObject.name] = errorObject.reason;
  });

  return result;
};
