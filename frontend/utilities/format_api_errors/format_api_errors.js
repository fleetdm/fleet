export default (error) => {
  if (!error.response || !error.response.errors) {
    return undefined;
  }

  const { errors: errorsArray } = error.response;
  const result = {};

  errorsArray.forEach((errorObject) => {
    result[errorObject.name] = errorObject.reason;
  });

  return result;
};
