export default (error: any) => {
  if (!error.response || !error.response.errors) {
    return undefined;
  }

  const { errors: errorsArray } = error.response;
  const result: { [key: string]: any } = {};

  errorsArray.forEach((errorObject: any) => {
    result[errorObject.name] = errorObject.reason;
  });

  return result;
};
