interface IApiErrorResponse {
  response?: {
    errors?: Array<{ name: string; reason: string }>;
  };
}

export default (error: IApiErrorResponse) => {
  if (!error.response || !error.response.errors) {
    return undefined;
  }

  const { errors: errorsArray } = error.response;
  const result: { [key: string]: string } = {};

  errorsArray.forEach((errorObject) => {
    result[errorObject.name] = errorObject.reason;
  });

  return result;
};
